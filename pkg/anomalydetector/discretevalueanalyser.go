package anomalydetector

import (
	"fmt"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

type okkoCount struct {
	ok uint
	ko uint
}

type okkoByPodName map[string]okkoCount
type discreteValueAnalyser interface {
	doAnalysis() (okkoByPodName, error)
}

var _ AnomalyDetector = &DiscreteValueOutOfListAnalyser{}

//DiscreteValueOutOfListAnalyser anomalyDetector that check the ratio of good/bad value and return the pods that exceed a given threshold for that ratio
type DiscreteValueOutOfListAnalyser struct {
	api.DiscreteValueOutOfList
	selector  labels.Selector
	analyser  discreteValueAnalyser
	podLister kv1.PodNamespaceLister
	logger    *zap.Logger
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *DiscreteValueOutOfListAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	listOfPods, err := d.podLister.List(d.selector)
	if err != nil {
		return nil, fmt.Errorf("can't list pods, error:%v", err)
	}

	listOfPods, err = pod.PurgeNotReadyPods(listOfPods)
	if err != nil {
		return nil, fmt.Errorf("can't purge not ready pods, error:%v", err)
	}
	podByName := map[string]*kapiv1.Pod{}
	podWithNoTraffic := map[string]*kapiv1.Pod{}

	for _, p := range listOfPods {
		podByName[p.Name] = p
		traffic, _, err2 := labeling.IsPodTrafficLabelOkOrPause(p)
		if err2 != nil {
			return nil, err2
		}
		if !traffic {
			podWithNoTraffic[p.Name] = p
		}
	}

	result := []*kapiv1.Pod{}
	countersByPods, err := d.analyser.doAnalysis()
	if err != nil {
		return nil, err
	}

	d.logger.Sugar().Debugf("Number of PODs reporting metrics:%d\n", len(countersByPods))

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
