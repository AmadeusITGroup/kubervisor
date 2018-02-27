package labeling

import (
	"fmt"
	"strconv"
	"time"

	kv1 "k8s.io/api/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

const (
	LabelTraffic      = "traffic"
	LabelTrafficYes   = "yes"
	LabelTrafficNo    = "no"
	LabelTrafficPause = "pause"
)

const (
	AnnotationRetryMode  = "breaker/policy"
	AnnotationRetryAt    = "breaker/retryAt"
	AnnotationRetryCount = "breaker/retryCount"
)

func ReadRetryStrategyMode(pod *kv1.Pod) v1.RetryStrategyMode {
	if pod != nil {
		if value, ok := pod.Annotations[AnnotationRetryMode]; ok {
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

func GetRetryAt(pod *kv1.Pod) (time.Time, error) {
	retryAt, ok := pod.Annotations[AnnotationRetryAt]
	if !ok {
		return time.Time{}, fmt.Errorf("No retryAt annotation ")
	}
	return time.Parse(time.RFC3339, retryAt)
}

func GetRetryCount(pod *kv1.Pod) (int, error) {
	retryCount, ok := pod.Annotations[AnnotationRetryCount]
	if !ok {
		return 0, fmt.Errorf("No retryCount annotation ")
	}
	return strconv.Atoi(retryCount)
}

func IsPodTrafficLabelOk(pod *kv1.Pod) (bool, bool, error) {
	trafficLabel, ok := pod.Labels[LabelTraffic]
	if !ok {
		return false, false, fmt.Errorf("No traffic label ")
	}
	return trafficLabel == LabelTrafficYes, trafficLabel == LabelTrafficPause, nil
}
