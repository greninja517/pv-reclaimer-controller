package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PVReclaimSpec defines the desired state of PVReclaim.
type PVReclaimSpec struct {

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	PersistentVolumeName string `json:"persistentVolumeName"`
}

// PVReclaimStatus defines the observed state of PVReclaim.
type PVReclaimPhase string

// for listing all the possible valid values
const (
	PendingPhase PVReclaimPhase = "Pending"
	SuccessPhase PVReclaimPhase = "Success"
	FailurePhase PVReclaimPhase = "Failed"
)

type PVReclaimStatus struct {
	Phase              PVReclaimPhase `json:"phase,omitempty"`
	ReclaimedTimestamp *metav1.Time   `json:"reclaimedTimestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=pvr
// PVReclaim is the Schema for the pvreclaims API.
type PVReclaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PVReclaimSpec   `json:"spec,omitempty"`
	Status PVReclaimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// PVReclaimList contains a list of PVReclaim.
type PVReclaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PVReclaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PVReclaim{}, &PVReclaimList{})
}
