package pod

import (
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
	kapiv1 "k8s.io/api/core/v1"
)

//FilterOut remove from the slice pods for which exclude function return true
func FilterOut(slice []*kapiv1.Pod, exclude func(*kapiv1.Pod) bool) []*kapiv1.Pod {
	b := []*kapiv1.Pod{}
	for _, x := range slice {
		if !exclude(x) {
			b = append(b, x)
		}
	}
	return b
}

//PurgeNotReadyPods keep only pods that are ready inside the slice
func PurgeNotReadyPods(pods []*kapiv1.Pod) []*kapiv1.Pod {
	return FilterOut(pods, func(a *kapiv1.Pod) bool { return !IsReady(a) })
}

//IsReady check if the pod is Ready
func IsReady(p *kapiv1.Pod) bool {
	if p.Status.Phase == kapiv1.PodRunning {
		for _, c := range p.Status.Conditions {
			if c.Type == kapiv1.PodReady {
				return c.Status == kapiv1.ConditionTrue
			}
		}
	}
	return false
}

//KeepRunningPods check if the pod is Ready
func KeepRunningPods(pods []*kapiv1.Pod) []*kapiv1.Pod {
	return FilterOut(pods, func(a *kapiv1.Pod) bool { return a.Status.Phase != kapiv1.PodRunning })
}

//KeepWithTrafficYesPods only keep pods marked to receive traffic. Does not mean that they actually receive any... just they are eligible. Does not mean either that the pod is Ready (probes to be checked)
func KeepWithTrafficYesPods(pods []*kapiv1.Pod) []*kapiv1.Pod {
	return FilterOut(pods, func(p *kapiv1.Pod) bool {
		if yes, _, _ := labeling.IsPodTrafficLabelOkOrPause(p); !yes {
			return true
		}
		return false
	})
}
