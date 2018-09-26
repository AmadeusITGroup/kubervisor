package labeling

import (
	"fmt"
	"strconv"
	"time"

	kv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

//Kubervisor keys for Labels and Annotations
const (
	LabelBreakerNameKey     = "kubervisor/name"
	LabelBreakerStrategyKey = "kubervisor/strategy"
	AnnotationBreakAtKey    = "breaker/breakAt"
	AnnotationRetryCountKey = "breaker/retryCount"
)

//GetBreakAt read the next retry time from Pod annotations
func GetBreakAt(pod *kv1.Pod) (time.Time, error) {
	if pod.Annotations == nil {
		return time.Time{}, fmt.Errorf("No breakAt annotation")
	}
	breakAt, ok := pod.Annotations[AnnotationBreakAtKey]
	if !ok {
		return time.Time{}, fmt.Errorf("No breakAt annotation")
	}
	return time.Parse(time.RFC3339, breakAt)
}

//GetRetryCount read the retry count from Pod annotations
func GetRetryCount(pod *kv1.Pod) (int, error) {
	if pod.Annotations == nil {
		return 0, nil
	}
	retryCount, ok := pod.Annotations[AnnotationRetryCountKey]
	if !ok {
		return 0, nil
	}
	return strconv.Atoi(retryCount)
}

//SelectorWithBreakerName augment the given selector with the breaker name
func SelectorWithBreakerName(inputSelector labels.Selector, breakerName string) (labels.Selector, error) {
	augmentedSelector := labels.Everything()
	if inputSelector != nil {
		augmentedSelector = inputSelector.DeepCopySelector()
	}

	var err error
	rqBreaker, err := labels.NewRequirement(LabelBreakerNameKey, selection.Equals, []string{breakerName})
	if err != nil {
		return nil, err
	}
	return augmentedSelector.Add(*rqBreaker), nil
}
