package v1

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
//+genclient:nonNamespaced
//+k8s:openapi-gen=true
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

type Health struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HealthSpec `json:"spec,omitempty"`
	Status            Status     `json:"status,omitempty"`
}

type HealthSpec struct {
	Interval           time.Duration       `json:"interval,omitempty"`
	ComponentSelectors map[string]string   `json:"componentSelectors,omitempty"`
	ServiceProbes      map[string]v1.Probe `json:"serviceProbes,omitempty"`
}

type Status struct {
	Components map[string]Component `json:"components,omitempty"`
}

type Component struct {
	Name    string      `json:"name,omitempty"`
	PodList []PodStatus `json:"podList,omitempty"`
	Message string      `json:"message,omitempty"`
}

type PodStatus struct {
	Name              string      `json:"name,omitempty"`
	Namespace         string      `json:"namespace,omitempty"`
	RestartCount      int         `json:"restartCount,omitempty"`
	Status            string      `json:"status,omitempty"`
	CreationTimestamp metav1.Time `json:"creationTimestamp,omitempty"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true

type HealthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Health `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Health{}, &HealthList{})
}
