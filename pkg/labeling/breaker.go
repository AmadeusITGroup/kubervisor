package labeling

import (
	"fmt"
	"strconv"
	"time"

	kv1 "k8s.io/api/core/v1"
)

//Kubervisor keys for Labels and Annotations
const (
	LabelBreakerNameKey     = "kubervisor/name"
	AnnotationBreakAtKey    = "breaker/breakAt"
	AnnotationRetryCountKey = "breaker/retryCount"
)

//GetRetryAt read the next retry time from Pod annotations
func GetRetryAt(pod *kv1.Pod) (time.Time, error) {
	retryAt, ok := pod.Annotations[AnnotationBreakAtKey]
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
