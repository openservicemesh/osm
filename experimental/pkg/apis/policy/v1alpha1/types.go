package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Backpressure is ...
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Backpressure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BackpressureSpec `json:"spec"`
}

// BackpressureSpec is ...
type BackpressureSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// MaxConnections is the max number of connections a proxy will make to the remote service.
	MaxConnections     int `json:"maxConnections,omitempty"`
	MaxRequests        int `json:"maxRequests,omitempty"`
	MaxPendingRequests int `json:"maxPendingRequests,omitempty"`
	MaxRetries         int `json:"maxRetries,omitempty"`
	MaxConnectionPools int `json:"maxConnectionPools,omitempty"`
}

// UnmarshalJSON is ...
func (spec *BackpressureSpec) UnmarshalJSON(data []byte) error {
	spec.MaxConnections = -1     // set default value before unmarshaling
	spec.MaxRequests = -1        // set default value before unmarshaling
	spec.MaxPendingRequests = -1 // set default value before unmarshaling
	spec.MaxRetries = -1         // set default value before unmarshaling
	spec.MaxConnectionPools = -1 // set default value before unmarshaling
	type Alias BackpressureSpec  // create alias to prevent endless loop
	tmp := (*Alias)(spec)

	return json.Unmarshal(data, tmp)
}

// BackpressureList is ...
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BackpressureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is the list of Backpressure
	Items []Backpressure `json:"items"`
}
