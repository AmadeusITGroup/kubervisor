package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BreakerConfig represents a Breaker configuration
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BreakerConfig struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired BreakerConfig specification
	Spec BreakerConfigSpec `json:"spec,omitempty"`

	// Status represents the current BreakerConfig status
	Status BreakerConfigStatus `json:"status,omitempty"`
}

// BreakerConfigList implements list of BreakerConfig.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BreakerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of BreakerConfig
	Items []BreakerConfig `json:"items"`
}

// BreakerConfigSpec contains BreakerConfig specification
type BreakerConfigSpec struct {
	Breaker  BreakerStrategy   `json:"breaker"`
	Retry    RetryStrategy     `json:"retry"`
	Selector map[string]string `json:"selector,omitempty"`
}

// BreakerConfigStatus contains BreakerConfig status
type BreakerConfigStatus struct {
	CurrentStatus string `json:"status"`
}

// BreakerStrategy contains BreakerStrategy definition
type BreakerStrategy struct {
	Mode            BreakerStrategyMode `json:"mode"`
	OkPromQlRequest string              `json:"okRequest"`
	KoPromQlRequest string              `json:"koRequest"`
}

// BreakerStrategyMode represent the breaker Strategy type
type BreakerStrategyMode string

const (
	// DefaultBreakerStrategy represent the default breaker strategy
	DefaultBreakerStrategy BreakerStrategyMode = "default"
)

// RetryStrategy contains RetryStrategy definition
type RetryStrategy struct {
	Mode        RetryStrategyMode `json:"mode"`
	RetryPeriod time.Duration     `json:"retryPeriod"`
}

// RetryStrategyMode represent the breaker Strategy Mode
type RetryStrategyMode string

const (
	// DefaultRetryStrategy represent the default retry strategy
	DefaultRetryStrategy RetryStrategyMode = "default"
)
