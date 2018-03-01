package anomalydetector

import (
	"fmt"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
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

//AnomalyDetector returns the list of pods that do not behave correctly according to the configuration
type AnomalyDetector interface {
	GetPodsOutOfBounds() ([]*kapiv1.Pod, error)
}

//DiscreteValueOutOfListAnalyser anomalyDetector that check the ratio of good/bad value and return the pods that exceed a given threshold for that ratio
type DiscreteValueOutOfListAnalyser struct {
	v1.DiscreteValueOutOfList
	selector    labels.Selector
	podAnalyser podAnalyser
	podLister   kv1.PodLister
	logger      *zap.Logger
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *DiscreteValueOutOfListAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	result := []*kapiv1.Pod{}
	countersByPods, err := d.podAnalyser.doAnalysis()
	if err != nil {
		return nil, err
	}

	d.logger.Sugar().Debugf("Number of PODs reporting metrics:%d\n", len(countersByPods))
	listOfPods, err := d.podLister.List(d.selector)
	if err != nil {
		return nil, fmt.Errorf("can't list pods")
	}

	listOfPods = purgeNotReadyPods(listOfPods)
	podByName := map[string]*kapiv1.Pod{}
	podWithNoTraffic := map[string]*kapiv1.Pod{}

	for _, p := range listOfPods {
		podByName[p.Name] = p
		if traffic, _, _ := labeling.IsPodTrafficLabelOkOrPause(p); !traffic {
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
		if sum >= *d.MinimumActivityCount {
			ratio := counter.ko * 100 / sum
			if ratio > *d.TolerancePercent {
				if p, ok := podByName[podName]; ok {
					// Only keeping known pod with ratio superior to Tolerance
					result = append(result, p)
				}
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
