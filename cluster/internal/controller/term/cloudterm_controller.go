/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package term
package term

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	termv1 "cluster/api/term/v1"
)

// CloudTermReconciler reconciles a CloudTerm object
type CloudTermReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Tracer   trace.Tracer
}

// +kubebuilder:rbac:groups=term.cts.io,resources=cloudterms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=term.cts.io,resources=cloudterms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=term.cts.io,resources=cloudterms/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *CloudTermReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 1. 获取 CloudTerm
	ct := &termv1.CloudTerm{}
	if err := r.Get(ctx, req.NamespacedName, ct); err != nil {
		if errors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{}, nil
		}
		r.Recorder.Event(ct, "Warning", "GetCloudTermFailed", err.Error())
		logf.FromContext(ctx).Error(err, "unable to fetch CloudTerm", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(ct.Annotations))
	ctx, span := r.Tracer.Start(ctx, "ReconcileCloudTerm")
	defer span.End()
	span.SetAttributes(
		attribute.String("CloudTerm.name", req.Name),
		attribute.String("CloudTerm.namespace", req.Namespace),
	)

	// 2. 初始化状态
	oldStatus := ct.Status.DeepCopy()
	if ct.Status.Phase == "" {
		ct.Status.Phase = termv1.CloudTermPending
		ct.Status.Replicas = 1
		ct.Status.NetworkStatuses = make([]*termv1.NetworkStatus, 0)
		ct.Status.Conditions = make([]metav1.Condition, 0)
	}
	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		log := logf.FromContext(ctx).WithValues("trace_id", sc.TraceID().String())
		ctx = logf.IntoContext(ctx, log)
	}

	// 2. 协调资源
	if ct.Spec.Replicas != nil {
		ct.Status.Replicas = *ct.Spec.Replicas
	}
	if err := r.reconcileStatefulSet(ctx, ct); err != nil {
		logf.FromContext(ctx).Error(err, "unable to put sts", "cloudterm", *ct)
		return ctrl.Result{}, r.updateStatusIfNeedWithCondition(ctx, oldStatus, ct, err, metav1.Condition{
			Type:    termv1.CloudTermConditionTypeStsReady,
			Status:  metav1.ConditionFalse,
			Reason:  "StatefulSetError",
			Message: err.Error(),
		})
	}
	meta.SetStatusCondition(&ct.Status.Conditions, metav1.Condition{
		Type:   termv1.CloudTermConditionTypeStsReady,
		Status: metav1.ConditionTrue,
		Reason: "Success",
	})
	// 3. 更新状态
	return r.updateNetworkStatus(ctx, oldStatus, ct)
}

func (r *CloudTermReconciler) updateNetworkStatus(ctx context.Context, oldStatus *termv1.CloudTermStatus, ct *termv1.CloudTerm) (ctrl.Result, error) {
	ctx, span := r.Tracer.Start(ctx, "updateNetworkStatus")
	defer span.End()

	if ct.Status.Replicas == 0 {
		ct.Status.Phase = termv1.CloudTermStop
		ct.Status.NetworkStatuses = make([]*termv1.NetworkStatus, 0)
		return ctrl.Result{}, r.updateStatusIfNeed(ctx, oldStatus, ct)
	}
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-0", ct.Name), Namespace: ct.Namespace}, pod); err != nil {
		if !errors.IsNotFound(err) {
			r.Recorder.Event(ct, "Warning", "GetPodFailed", err.Error())
			span.RecordError(err)
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	if pod.Status.Phase != corev1.PodRunning {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// 从 Pod 获取端口信息 (假设端口信息在注解中)
	annotations := pod.GetAnnotations()
	portStatus, ok := annotations["wooshnet/portstatus"]
	if !ok {
		r.Recorder.Eventf(ct, "Warning", "PortStatusNotFound", "Pod %s-0 wooshnet/portstatus not found", ct.Name)
		span.RecordError(fmt.Errorf("pod %s-0 wooshnet/portstatus not found", ct.Name))
		ct.Status.Phase = termv1.CloudTermError
		return ctrl.Result{}, r.updateStatusIfNeedWithCondition(ctx, oldStatus, ct, nil, metav1.Condition{
			Type:    termv1.CloudTermConditionTypePortReady,
			Status:  metav1.ConditionFalse,
			Reason:  "PortStatusMissing",
			Message: "Annotation wooshnet/portstatus not found",
		})
	}
	var currentStatus []*termv1.PortStatus
	if err := json.Unmarshal([]byte(portStatus), &currentStatus); err != nil {
		ct.Status.Phase = termv1.CloudTermError
		span.RecordError(err)
		return ctrl.Result{}, r.updateStatusIfNeedWithCondition(ctx, oldStatus, ct, err, metav1.Condition{
			Type:    termv1.CloudTermConditionTypePortReady,
			Status:  metav1.ConditionFalse,
			Reason:  "PortStatusInvalid",
			Message: fmt.Sprintf("Failed to parse: %v", err),
		})
	}
	var attached bool
	networkStatuses := make([]*termv1.NetworkStatus, 0, len(currentStatus))
	for _, s := range currentStatus {
		for _, es := range ct.Spec.Networks {
			if es.NetworkID == s.NetworkID {
				networkStatuses = append(networkStatuses, &termv1.NetworkStatus{
					PortStatus: *s,
					EIP:        es.EIP,
				})
				if es.EIP != "" {
					attached = true
				}
				break
			}
		}
	}
	ct.Status.NetworkStatuses = networkStatuses
	if attached {
		ct.Status.Phase = termv1.CloudTermRunning
	} else {
		ct.Status.Phase = termv1.CloudTermReady
	}
	return ctrl.Result{}, r.updateStatusIfNeed(ctx, oldStatus, ct)
}

func (r *CloudTermReconciler) reconcileVolume(ctx context.Context, ct *termv1.CloudTerm, volume *termv1.Volume) error {
	ctx, span := r.Tracer.Start(ctx, "reconcileVolume")
	defer span.End()

	storage := resource.NewQuantity(volume.Size*1024*1024*1024, resource.BinarySI)
	pv := &corev1.PersistentVolume{}
	if volume.PersistentVolumeSource != nil {
		data, err := json.Marshal(volume.PersistentVolumeSource)
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to marshal volume source: %w", err)
		}
		md5Writer := md5.New()
		_, err = md5Writer.Write(data)
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to write data to md5 writer: %w", err)
		}

		pv.Name = fmt.Sprintf("cloudterm-%x", md5Writer.Sum(nil))
		pv.Labels = map[string]string{
			"app.kubernetes.io/name": "cloudterm",
			"cloudterm/name":         pv.Name,
		}
		volumeMode := corev1.PersistentVolumeFilesystem
		op, err := controllerutil.CreateOrPatch(ctx, r.Client, pv, func() error {
			if pv.CreationTimestamp.IsZero() {
				pv.Spec = corev1.PersistentVolumeSpec{
					Capacity: corev1.ResourceList{
						"storage": *storage,
					},
					PersistentVolumeSource: *volume.PersistentVolumeSource,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					StorageClassName:              pv.Name,
					VolumeMode:                    &volumeMode,
				}
				if err := ctrl.SetControllerReference(ct, pv, r.Scheme); err != nil {
					return fmt.Errorf("PersistentVolume %s unable to set controller reference, error: %w", pv.Name, err)
				}
			}
			return nil
		})
		if err != nil {
			r.Recorder.Eventf(ct, "Warning", "PVReconcileFailed", "Operation %s failed: %v", op, err)
			span.RecordError(err)
			return fmt.Errorf("failed to reconcile PV %s: %w", pv.Name, err)
		}
	} else {
		if err := r.Get(ctx, types.NamespacedName{Name: volume.VolumeMount.Name}, pv); err != nil {
			r.Recorder.Event(ct, "Warning", "GetPVFailed", err.Error())
			span.RecordError(err)
			return fmt.Errorf("failed to get pv %s: %w", volume.VolumeMount.Name, err)
		}
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      volume.VolumeMount.Name,
			Namespace: ct.Namespace,
			Labels:    ct.Labels,
		},
	}
	op, err := controllerutil.CreateOrPatch(ctx, r.Client, pvc, func() error {
		if pvc.CreationTimestamp.IsZero() {
			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: pv.Labels,
				},
				StorageClassName: &pv.Spec.StorageClassName,
				Resources: corev1.VolumeResourceRequirements{
					Requests: pv.Spec.Capacity,
				},
			}
			if err := ctrl.SetControllerReference(ct, pvc, r.Scheme); err != nil {
				return fmt.Errorf("PersistentVolumeClaim %s unable to set controller reference, error: %w", volume.VolumeMount.Name, err)
			}
		}
		return nil
	})
	if err != nil {
		r.Recorder.Eventf(ct, "Warning", "PVCReconcileFailed", "Operation %s failed: %v", op, err)
		span.RecordError(err)
		return fmt.Errorf("failed to reconcile PVC %s: %w", volume.VolumeMount.Name, err)
	}
	return nil
}

func (r *CloudTermReconciler) reconcileStatefulSet(ctx context.Context, ct *termv1.CloudTerm) error {
	ctx, span := r.Tracer.Start(ctx, "reconcileStatefulSet")
	defer span.End()

	ports := make([]*termv1.Port, 0, len(ct.Spec.Networks))
	for _, n := range ct.Spec.Networks {
		ports = append(ports, &n.Port)
	}
	networkJSON, err := json.Marshal(ports)
	if err != nil {
		r.Recorder.Event(ct, "Warning", "MarshalNetworksFailed", err.Error())
		span.RecordError(err)
		return fmt.Errorf("failed to marshal networks: %w", err)
	}
	// Get pod spec from ConfigMap
	podSpec, err := r.getPodSpecWithConfigMap(ctx, ct)
	if err != nil {
		return err
	}
	// Apply custom container config
	if err = r.applyContainerConfig(ctx, ct, podSpec); err != nil {
		return err
	}
	// Prepare StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ct.Name,
			Namespace: ct.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "cloudterm",
				"app.kubernetes.io/part-of": "cloudterm-operator",
			},
		},
	}
	_, err = controllerutil.CreateOrPatch(ctx, r.Client, sts, func() error {
		if sts.CreationTimestamp.IsZero() {
			labels := map[string]string{
				"app.kubernetes.io/name": "cloudterm",
				"cloudterm/name":         ct.Name,
			}
			sts.Spec = appsv1.StatefulSetSpec{
				Replicas: &ct.Status.Replicas,
				Selector: &metav1.LabelSelector{MatchLabels: labels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
						Annotations: map[string]string{
							"v1.multus-cni.io/default-network": "wooshnet",
							"wooshnet/ports":                   string(networkJSON),
						},
					},
					Spec: *podSpec,
				},
			}
		} else {
			// Only update mutable fields
			sts.Spec.Replicas = &ct.Status.Replicas
			sts.Spec.Template.Annotations["wooshnet/ports"] = string(networkJSON)
			sts.Spec.Template.Spec = *podSpec
		}
		if err := ctrl.SetControllerReference(ct, sts, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}
		return nil
	})
	if err != nil {
		r.Recorder.Event(ct, "Warning", "StatefulSetReconcileFailed", err.Error())
		span.RecordError(err)
		return fmt.Errorf("failed to reconcile StatefulSet: %w", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudTermReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("term-cloudterm").
		For(&termv1.CloudTerm{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
