package v1

import (
	"time"

	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubervisorService represents a Breaker configuration
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubervisorService struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired KubervisorService specification
	Spec KubervisorServiceSpec `json:"spec,omitempty"`

	// Status represents the current KubervisorService status
	Status KubervisorServiceStatus `json:"status,omitempty"`
}

// KubervisorServiceList implements list of KubervisorService.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubervisorServiceList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of KubervisorService
	Items []KubervisorService `json:"items"`
}

// KubervisorServiceSpec contains KubervisorService specification
type KubervisorServiceSpec struct {
	Breaker   BreakerStrategy   `json:"breaker"`
	Activator ActivatorStrategy `json:"activator"`
	Service   string            `json:"service,omitempty"`
}

// KubervisorServiceConditionType KubervisorService Condition Type
type KubervisorServiceConditionType string

// These are valid conditions of a KubervisorService.
const (
	// KubervisorServiceInitFailed means the KubervisorService has completed its execution.
	KubervisorServiceInitFailed KubervisorServiceConditionType = "InitFailed"
	// KubervisorServiceRunning means the KubervisorService has completed its execution.
	KubervisorServiceRunning KubervisorServiceConditionType = "Running"
	// KubeServiceNotAvailable means the KubervisorService has completed its execution.
	KubeServiceNotAvailable KubervisorServiceConditionType = "ServiceNotAvailable"
	// KubervisorServiceFailed means the KubervisorService has failed its execution.
	KubervisorServiceFailed KubervisorServiceConditionType = "Failed"
)

// KubervisorServiceCondition represent the condition of the KubervisorService
type KubervisorServiceCondition struct {
	// Type of KubervisorService condition
	Type KubervisorServiceConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status api.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transited from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// KubervisorServiceStatus contains KubervisorService status
type KubervisorServiceStatus struct {
	// Conditions represent the latest available observations of an object's current state.
	Conditions []KubervisorServiceCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// StartTime represents time when the KubervisorService was acknowledged by the Kubervisor controller
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	// StartTime doesn't consider startime of `ExternalReference`
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Status represent the breaker status: contains status by pods (score, breaked or not, reason)
	Breaker *BreakerStatus `json:"breaker,omitempty"`
}

// BreakerStatus contains breaker status
type BreakerStatus struct {
	NbPodsManaged uint32 `json:"nbPodsManaged,omitempty"`
	NbPodsBreaked uint32 `json:"nbPodsBreaked,omitempty"`
	NbPodsPaused  uint32 `json:"nbPodsPaused,omitempty"`
	NbPodsUnknown uint32 `json:"nbPodsUnknown,omitempty"`
	// Last time the condition was checked.
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
}

// BreakerStrategy contains BreakerStrategy definition
type BreakerStrategy struct {
	EvaluationPeriod      time.Duration `json:"evaluationPeriod,omitempty"`
	MinPodsAvailableCount *uint         `json:"minPodsAvailableCount,omitempty"`
	MinPodsAvailableRatio *uint         `json:"minPodsAvailableRatio,omitempty"`

	DiscreteValueOutOfList *DiscreteValueOutOfList `json:"discreteValueOutOfList,omitempty"`

	CustomService string `json:"customService,omitempty"`
}

// DiscreteValueOutOfList detect anomaly when the a value is not in the list with a ratio that exceed the tolerance
// The promQL should return counter that are grouped by:
// 1-the key of the value to monitor
// 2-the podname
type DiscreteValueOutOfList struct {
	PrometheusService    string   `json:"prometheusService"`
	PromQL               string   `json:"promQL"`               // example: sum(delta(ms_rpc_count{job=\"kubernetes-pods\",run=\"foo\"}[10s])) by (code,kubernetes_pod_name)
	Key                  string   `json:"key"`                  // Key for the metrics. For the previous example it will be "code"
	PodNameKey           string   `json:"podNamekey"`           // Key to access the podName
	GoodValues           []string `json:"goodValues,omitempty"` // Good Values ["200","201"]. If empty means that BadValues should be used to do exclusion instead of inclusion.
	BadValues            []string `json:"badValues,omitempty"`  // Bad Values ["500","404"].
	TolerancePercent     *uint    `json:"tolerance"`            // % of Bad values tolerated until the pod is considered out of SLA
	MinimumActivityCount *uint    `json:"minActivity"`          // Minimum number of event required to perform analysis on the pod

}

// ActivatorStrategy contains ActivatorStrategy definition
type ActivatorStrategy struct {
	Mode          ActivatorStrategyMode `json:"mode"`
	Period        time.Duration         `json:"period,omitempty"`
	MaxRetryCount *uint                 `json:"maxRetryCount,omitempty"`
	MaxPauseCount *uint                 `json:"maxPauseCount,omitempty"`
}

// ActivatorStrategyMode represent the breaker Strategy Mode
type ActivatorStrategyMode string

// ActivatorStrategyMode defines the possible behavior of the activator
const (
	ActivatorStrategyModePeriodic      ActivatorStrategyMode = "periodic"
	ActivatorStrategyModeRetryAndKill  ActivatorStrategyMode = "retryAndKill"
	ActivatorStrategyModeRetryAndPause ActivatorStrategyMode = "retryAndPause"
)
