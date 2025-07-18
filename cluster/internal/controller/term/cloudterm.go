package term

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	termv1 "cluster/api/term/v1"
)

const (
	cloudTermMainAppContainerName = "cloudterm"
)

func (r *CloudTermReconciler) getPodSpecWithConfigMap(ctx context.Context, ct *termv1.CloudTerm) (*corev1.PodSpec, error) {
	ctx, span := r.Tracer.Start(ctx, "getPodSpecWithConfigMap")
	defer span.End()
	cmNs := types.NamespacedName{Name: ct.Spec.PodSpecConfig, Namespace: ct.Namespace}
	if strings.Contains(ct.Spec.PodSpecConfig, "/") {
		cmNames := strings.SplitN(ct.Spec.PodSpecConfig, "/", 2)
		cmNs = types.NamespacedName{Name: cmNames[1], Namespace: cmNames[0]}
	}
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, cmNs, cm); err != nil {
		r.Recorder.Event(ct, "Warning", "GetConfigMapFailed", err.Error())
		span.RecordError(err)
		return nil, fmt.Errorf("ConfigMap %s unable to fetch, error: %w", ct.Spec.PodSpecConfig, err)
	}
	podSpec := &corev1.PodSpec{}
	err := yaml.Unmarshal([]byte(cm.Data["podSpec"]), &podSpec)
	if err != nil {
		r.Recorder.Event(ct, "Warning", "UnmarshalPodSpecFailed", err.Error())
		span.RecordError(err)
		return nil, fmt.Errorf("PodSpec %s unable to unmarshal, error: %w", ct.Spec.PodSpecConfig, err)
	}
	return podSpec, nil
}

// 应用容器配置的辅助函数
func (r *CloudTermReconciler) applyContainerConfig(ctx context.Context, ct *termv1.CloudTerm, podSpec *corev1.PodSpec) error {
	ctx, span := r.Tracer.Start(ctx, "applyContainerConfig")
	defer span.End()

	if err := r.applyProperty(ctx, ct, podSpec); err != nil {
		return err
	}
	// 挂载安卓属性配置文件
	for i, container := range podSpec.Containers {
		for _, inputContainer := range ct.Spec.Containers {
			if inputContainer.Name == container.Name {
				// 更新镜像和资源
				podSpec.Containers[i].Image = inputContainer.Image
				podSpec.Containers[i].Resources = corev1.ResourceRequirements{
					Requests: inputContainer.Requests,
					Limits:   inputContainer.Requests, // 使用与请求相同的值作为限制
				}
				// 更新端口
				if len(inputContainer.Ports) > 0 {
					podSpec.Containers[i].Ports = inputContainer.Ports
				}
				for j, volume := range inputContainer.Volumes {
					if err := r.reconcileVolume(ctx, ct, volume); err != nil {
						return err
					}
					if !hasMountPath(container.VolumeMounts, volume.VolumeMount.MountPath) {
						podSpec.Containers[i].VolumeMounts = append(podSpec.Containers[i].VolumeMounts, inputContainer.Volumes[j].VolumeMount)
					}
					if !hasVolume(podSpec.Volumes, volume.VolumeMount.Name) {
						podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
							Name: volume.VolumeMount.Name,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: volume.VolumeMount.Name,
								},
							},
						})
					}
				}
				// 判断是否是安卓主应用容器 TODO: 是否有更好的判断方式
				if ct.Spec.Property != nil && *ct.Spec.Property != "" && podSpec.Containers[i].Name == cloudTermMainAppContainerName {
					podSpec.Containers[i].VolumeMounts = append(podSpec.Containers[i].VolumeMounts, corev1.VolumeMount{
						Name:      "cloudterm-local-prop",
						MountPath: "/data/local.prop",
						SubPath:   "local.prop",
					})
				}
			}
		}
	}
	return nil
}

// 辅助函数：检查卷挂载
func hasMountPath(mounts []corev1.VolumeMount, path string) bool {
	for _, m := range mounts {
		if m.MountPath == path {
			return true
		}
	}
	return false
}

// 辅助函数：检查卷
func hasVolume(volumes []corev1.Volume, name string) bool {
	for _, v := range volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (r *CloudTermReconciler) updateStatusIfNeed(ctx context.Context, oldStatus *termv1.CloudTermStatus, ct *termv1.CloudTerm) error {
	// skip status update if the status is not exact same
	if apiequality.Semantic.DeepEqual(oldStatus, ct.Status) {
		return nil
	}
	if err := r.Status().Update(ctx, ct); err != nil {
		r.Recorder.Event(ct, corev1.EventTypeWarning, "FailedUpdateStatus", err.Error())
		logf.FromContext(ctx).Error(err, "unable to update CloudTerm status", "cloudterm", *ct)
		return fmt.Errorf("failed to update status for %s: %w", ct.Name, err)
	}
	return nil
}

func (r *CloudTermReconciler) updateStatusIfNeedWithCondition(ctx context.Context, oldStatus *termv1.CloudTermStatus, ct *termv1.CloudTerm, err error, condition metav1.Condition) error {
	meta.SetStatusCondition(&ct.Status.Conditions, condition)
	if updateErr := r.updateStatusIfNeed(ctx, oldStatus, ct); updateErr != nil {
		return updateErr
	}
	return err
}

func (r *CloudTermReconciler) applyProperty(ctx context.Context, ct *termv1.CloudTerm, podSpec *corev1.PodSpec) error {
	ctx, span := r.Tracer.Start(ctx, "applyProperty")
	defer span.End()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("cloudterm-%s-local-prop", ct.Name),
			Namespace: ct.Namespace,
		},
	}
	if ct.Spec.Property == nil || *ct.Spec.Property == "" {
		if err := r.Delete(ctx, cm); err != nil {
			if !errors.IsNotFound(err) {
				r.Recorder.Event(ct, "Warning", "DeleteConfigMapFailed", err.Error())
				span.RecordError(err)
				return fmt.Errorf("failed to delete ConfigMap: %w", err)
			}
		}
		return nil
	}
	_, err := controllerutil.CreateOrPatch(ctx, r.Client, cm, func() error {
		if cm.CreationTimestamp.IsZero() {
			cm.Data = map[string]string{
				"local.prop": *ct.Spec.Property,
			}
			if err := ctrl.SetControllerReference(ct, cm, r.Scheme); err != nil {
				return fmt.Errorf("ConfigMap %s unable to set controller reference, error: %w", cm.Name, err)
			}
		} else {
			cm.Data["local.prop"] = *ct.Spec.Property
		}
		return nil
	})
	if err != nil {
		r.Recorder.Event(ct, "Warning", "CreateConfigMapFailed", err.Error())
		span.RecordError(err)
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}
	defaultMode := int32(0400)
	podSpec.Volumes = append(podSpec.Volumes,
		corev1.Volume{
			Name: "cloudterm-local-prop",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
					DefaultMode: &defaultMode,
					Items: []corev1.KeyToPath{
						{
							Key:  "local.prop",
							Path: "local.prop",
						},
					},
				},
			},
		})
	return nil
}
