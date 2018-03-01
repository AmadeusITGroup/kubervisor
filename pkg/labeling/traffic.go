package labeling

import (
	"fmt"

	kv1 "k8s.io/api/core/v1"
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
	LabelTrafficKey = "kubervisor/traffic"
)

//SetTraficLabel create or update the value of the label LabelTrafficKey
func SetTraficLabel(pod *kv1.Pod, val LabelTraffic) {
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[LabelTrafficKey] = string(val)
}

//IsPodTrafficLabelOkOrPause check if the pod is marked to receive traffic or is in pause
func IsPodTrafficLabelOkOrPause(pod *kv1.Pod) (bool, bool, error) {
	trafficLabel, ok := pod.Labels[LabelTrafficKey]
	if !ok {
		return false, false, fmt.Errorf("No traffic label ")
	}
	if l, err := ToLabelTraffic(trafficLabel); err != nil {
		return false, false, err
	} else {
		return l == LabelTrafficYes, l == LabelTrafficPause, nil
	}

}

//ToLabelTraffic check and convert to LabelTraffic
func ToLabelTraffic(value string) (LabelTraffic, error) {
	switch value {
	case string(LabelTrafficYes):
		return LabelTrafficYes, nil
	case string(LabelTrafficNo):
		return LabelTrafficNo, nil
	case string(LabelTrafficPause):
		return LabelTrafficPause, nil
	default:
		return "", fmt.Errorf("Unknown value %s for LabelTraffic", value)
	}

}
