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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	CloudTermConditionTypeStsReady  = "StatefulSetReady"
	CloudTermConditionTypePortReady = "PortReady"

	// CloudTermPending means the CR has been created and that's the initial status
	CloudTermPending = "Pending"
	// CloudTermRunning means the CR is all ready and wait for port attach network
	CloudTermReady = "Ready"
	// CloudTermRunning means the CR is running and server is available
	CloudTermRunning = "Running"
	// CloudTermError means the CR is Error
	CloudTermError = "Error"
	// CloudTermStop means the CR is Stop
	CloudTermStop = "Stop"
)

// FixedIP defines a single fixed IP configuration.
type FixedIP struct {
	// SubnetID is the UUID of the subnet to which this IP belongs.
	// +kubebuilder:validation:Required
	SubnetID string `json:"subnet_id"`

	// IPAddress is the allocated fixed IP (optional).
	// +optional
	IPAddress string `json:"ip_address,omitempty"`

	// Gateway is the gateway address for this subnet (optional).
	// +optional
	Gateway string `json:"gateway,omitempty"`

	// Cidr is the CIDR notation for the subnet (optional).
	// +optional
	Cidr string `json:"cidr,omitempty"`

	// Visible indicates whether the IP should be exposed externally (optional).
	// +optional
	Visible bool `json:"visible,omitempty"`
}

// Network represents a complete network configuration (Port + optional EIP).
type Network struct {
	Port `json:",inline"`

	// EIP is the elastic public IP bound to this network (optional).
	// +optional
	EIP string `json:"eip,omitempty"`
}

// Port describes a Neutron/OpenStack port configuration.
type Port struct {
	// NetworkID is the UUID of the network this port belongs to.
	// +kubebuilder:validation:Required
	NetworkID string `json:"network_id"`

	// FixedIPs is a list of fixed IP configurations.
	// +optional
	FixedIPs []*FixedIP `json:"fixed_ips"`

	// SecurityGroups is a list of security group UUIDs (optional).
	// +optional
	SecurityGroups []string `json:"security_groups,omitempty"`
}

// NetworkStatus reflects runtime network status (PortStatus + optional EIP).
type NetworkStatus struct {
	PortStatus `json:",inline"`

	// EIP is the public IP actually bound (optional).
	// +optional
	EIP string `json:"eip,omitempty"`
}

// PortStatus contains runtime details returned by Neutron/OpenStack.
type PortStatus struct {
	// NetworkID is the UUID of the network.
	NetworkID string `json:"network_id"`

	// FixedIPs are the IPs assigned to this port.
	FixedIPs []*FixedIP `json:"fixed_ips"`

	// SecurityGroups are the applied security group UUIDs.
	SecurityGroups []string `json:"security_groups,omitempty"`

	// ID is the UUID of the port (optional).
	// +optional
	ID string `json:"id,omitempty"`

	// AdminStateUp indicates administrative state (optional).
	// +optional
	AdminStateUp bool `json:"admin_state_up,omitempty"`

	// Status is the operational status string (optional).
	// +optional
	Status string `json:"status,omitempty"`

	// MacAddress is the assigned MAC address (optional).
	// +optional
	MacAddress string `json:"mac_address,omitempty"`

	// TenantID is the tenant identifier (optional).
	// +optional
	TenantID string `json:"tenant_id,omitempty"`

	// ProjectID is the project identifier (optional).
	// +optional
	ProjectID string `json:"project_id,omitempty"`

	// PortID is kept for backward compatibility (optional).
	// +optional
	PortID string `json:"portId,omitempty"`

	// PortReady indicates whether the port is ready (optional).
	// +optional
	PortReady bool `json:"PortReady,omitempty"`

	// Ifname is the host-side interface name (optional).
	// +optional
	Ifname string `json:"ifname,omitempty"`

	// Nicname is a custom NIC name (optional).
	// +optional
	Nicname string `json:"nicname,omitempty"`

	// Ifaceid is the interface identifier (optional).
	// +optional
	Ifaceid string `json:"ifaceid,omitempty"`

	// Interface is a human-readable interface description (optional).
	// +optional
	Interface string `json:"interface,omitempty"`

	// DeviceReady indicates whether the underlying device is ready (optional).
	// +optional
	DeviceReady bool `json:"deviceReady,omitempty"`

	// VifType is the virtual interface type (optional).
	// +optional
	VifType string `json:"vif_type,omitempty"`
}

// Volume bundles a Kubernetes VolumeMount with its underlying PV source.
type Volume struct {
	// VolumeMount describes the mount path and sub-path.
	VolumeMount corev1.VolumeMount `json:"volumeMount"`

	// PersistentVolumeSource is the underlying PV configuration (optional).
	// +optional
	PersistentVolumeSource *corev1.PersistentVolumeSource `json:"persistentVolumeSource,omitempty"`

	// Size is the requested capacity in bytes.
	// +kubebuilder:validation:Minimum=0
	Size int64 `json:"size"`
}

// Container describes a simplified application container.
type Container struct {
	// Name of the container.
	Name string `json:"name"`

	// Image is the container image to run.
	Image string `json:"image"`

	// Ports to expose from the container (optional).
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	// Requests are the resource requests (CPU/Memory).
	Requests corev1.ResourceList `json:"requests"`

	// Volumes are the volumes to mount (optional).
	// +optional
	Volumes []*Volume `json:"volumes,omitempty"`
}

// CloudTermSpec defines the desired state of CloudTerm.
type CloudTermSpec struct {
	// Replicas is the desired number of StatefulSet replicas (optional).
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`

	// Property is an optional opaque string for custom configuration.
	// +optional
	Property *string `json:"property,omitempty"`

	// Containers is the list of application containers to run.
	// +kubebuilder:validation:MinItems=1
	Containers []Container `json:"containers"`

	// Networks is the list of network configurations.
	// +optional
	Networks []*Network `json:"networks"`

	// PodSpecConfig is the name (or namespace/name) of a ConfigMap
	// that contains additional PodSpec overrides (optional).
	// +optional
	PodSpecConfig string `json:"podSpecConfig"`
}

// CloudTermStatus defines the observed state of CloudTerm.
type CloudTermStatus struct {
	// Phase is the high-level operational state.
	// +kubebuilder:validation:Enum=Pending;Ready;Running;Error;Stop
	Phase string `json:"phase"`

	// Replicas is the current number of running replicas.
	Replicas int32 `json:"replicas"`

	// NetworkStatuses reflects the observed network state for each configured network.
	// +optional
	NetworkStatuses []*NetworkStatus `json:"networkStatuses,omitempty"`

	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Addr",type=string,JSONPath=`.status.networkStatuses[*].fixed_ips[*].ip_address`
// +kubebuilder:printcolumn:name="EIP",type=string,JSONPath=`.status.networkStatuses[*].eip`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CloudTerm is the Schema for the cloudterms API.
type CloudTerm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudTermSpec   `json:"spec,omitempty"`
	Status CloudTermStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// CloudTermList contains a list of CloudTerm.
type CloudTermList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudTerm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudTerm{}, &CloudTermList{})
}
