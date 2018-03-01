package labeling

import (
	"fmt"
	"strconv"
	"time"

	kv1 "k8s.io/api/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

//LabelTraffic are possible values associated to label with key LabelTrafficKey
type LabelTraffic string

//Possible values of LabelTraffic
const (
	LabelTrafficYes   LabelTraffic = "yes"
	LabelTrafficNo    LabelTraffic = "no"
	LabelTrafficPause LabelTraffic = "pause"
)

//Kubervisor keys for Labels and Annotations
const (
	LabelTrafficKey         = "kubervisor/traffic"
	AnnotationRetryModeKey  = "breaker/policy"
	AnnotationRetryAtKey    = "breaker/retryAt"
	AnnotationRetryCountKey = "breaker/retryCount"
)

//SetTraficLabel create or update the value of the label LabelTrafficKey
func SetTraficLabel(pod *kv1.Pod, val LabelTraffic) {
	pod.Labels[LabelTrafficKey] = string(val)
}

//GetRetryStrategyMode read the retry strategy from Pod annotations
func GetRetryStrategyMode(pod *kv1.Pod) v1.RetryStrategyMode {
	if pod != nil {
		if value, ok := pod.Annotations[AnnotationRetryModeKey]; ok {
			switch value {
			case string(v1.RetryStrategyModePeriodic):
				return v1.RetryStrategyModePeriodic
			case string(v1.RetryStrategyModeRetryAndKill):
				return v1.RetryStrategyModeRetryAndKill
			case string(v1.RetryStrategyModeRetryAndPause):
				return v1.RetryStrategyModeRetryAndPause
			}
		}
	}
	// default value
	return v1.RetryStrategyModeDisabled
}

//GetRetryAt read the next retry time from Pod annotations
func GetRetryAt(pod *kv1.Pod) (time.Time, error) {
	retryAt, ok := pod.Annotations[AnnotationRetryAtKey]
	if !ok {
		return time.Time{}, fmt.Errorf("No retryAt annotation ")
	}
	return time.Parse(time.RFC3339, retryAt)
}

//GetRetryCount read the retry count from Pod annotations
func GetRetryCount(pod *kv1.Pod) (int, error) {
	retryCount, ok := pod.Annotations[AnnotationRetryCountKey]
	if !ok {
		return 0, fmt.Errorf("No retryCount annotation ")
	}
	return strconv.Atoi(retryCount)
}

//IsPodTrafficLabelOkOrPause check if the pod is marked to receive traffic or is in pause
func IsPodTrafficLabelOkOrPause(pod *kv1.Pod) (bool, bool, error) {
	trafficLabel, ok := pod.Labels[LabelTrafficKey]
	if !ok {
		return false, false, fmt.Errorf("No traffic label ")
	}
	return trafficLabel == string(LabelTrafficYes), trafficLabel == string(LabelTrafficPause), nil
}
