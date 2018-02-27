package anomalyDetector

import (
	"fmt"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
)

type okkoCount struct {
	ok uint
	ko uint
}

type okkoByPodName map[string]okkoCount
type podAnalyser interface {
	doAnalysis() (okkoByPodName, error)
}

type AnomalyDetector interface {
	GetPodsOutOfBounds() ([]*kapiv1.Pod, error)
}

type DiscreteValueOutOfListAnalyser struct {
	v1.BreakerConfigSpec
	podAnalyser podAnalyser
	podLister   kv1.PodLister
	logger      *zap.Logger
}

func (d *DiscreteValueOutOfListAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	result := []*kapiv1.Pod{}
	countersByPods, err := d.podAnalyser.doAnalysis()
	if err != nil {
		return nil, err
	}

	d.logger.Sugar().Debugf("Number of PODs reporting metrics:%d\n", len(countersByPods))
	listOfPods, err := d.podLister.List(d.Selector)
	if err != nil {
		return nil, fmt.Errorf("can't list pods")
	}

	listOfPods = purgeNotReadyPods(listOfPods)
	podByName := map[string]*kapiv1.Pod{}
	podWithNoTraffic := map[string]*kapiv1.Pod{}

	for _, p := range listOfPods {
		podByName[p.Name] = p
		if traffic, _, _ := labeling.IsPodTrafficLabelOk(p); !traffic {
			podWithNoTraffic[p.Name] = p
		}
	}

	for podName, counter := range countersByPods {
		_, found := podWithNoTraffic[podName]
		if found {
			d.logger.Sugar().Infof("the pod %s metrics are ignored now has it is marked out of traffic\n", podName)
			continue
		}

		sum := counter.ok + counter.ko
		if sum != 0 {
			e := counter.ko * 100 / sum
			if e > d.Breaker.DiscreteValueOutOfList.TolerancePercent {
				//fmt.Printf("[%s] POD with above error threshold [ %d > %d ]: %s\n", runLabelValue, int(e), int(errorRatioThreshold), podName)
				if p, ok := podByName[podName]; !ok {
					//fmt.Printf("[%s] Pod %s reported by prometheus is not under informer. Must have been deleted.\n", runLabelValue, podName)
				} else {
					//aboveThreshold[podName] = p
					result = append(result, p)
				}
			} else if e > 0 {
				//fmt.Printf("[%s] POD with error but bellow threshold [ %d < %d ]: %s\n", runLabelValue, int(e), int(errorRatioThreshold), podName)
			}
		}
	}
	return result, nil
}

func purgeNotReadyPods(pods []*kapiv1.Pod) []*kapiv1.Pod {
	result := []*kapiv1.Pod{}
podLoop:
	for _, p := range pods {
		if p.Status.Phase == kapiv1.PodRunning {
			for _, c := range p.Status.Conditions {
				if c.Type == kapiv1.PodReady {
					if c.Status != kapiv1.ConditionTrue {
						continue podLoop
					}
				}
			}
			result = append(result, p)
		}
	}
	return result
}
