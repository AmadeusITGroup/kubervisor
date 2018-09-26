package anomalydetector

import (
	"fmt"
	"math"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

var _ AnomalyDetector = &ContinuousValueDeviationAnalyser{}

//deviationByPodName float64: 1=no deviation at all, 0.2=80% deviation down, 1.7=70% deviation up
type deviationByPodName map[string]float64
type continuousValueAnalyser interface {
	doAnalysis() (deviationByPodName, error)
}

//ContinuousValueDeviationAnalyser anomalyDetector that check the deviation of a continous value compare to average
type ContinuousValueDeviationAnalyser struct {
	api.ContinuousValueDeviation
	selector  labels.Selector
	analyser  continuousValueAnalyser
	podLister kv1.PodNamespaceLister
	logger    *zap.Logger
}

//GetPodsOutOfBounds implements interface AnomalyDetector
func (d *ContinuousValueDeviationAnalyser) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
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
	deviationByPods, err := d.analyser.doAnalysis()
	if err != nil {
		return nil, err
	}
	d.logger.Sugar().Debugf("Number of PODs reporting metrics:%d\n", len(deviationByPods))

	if len(deviationByPods) == 0 {
		return result, nil
	}

	maxDeviation := *d.ContinuousValueDeviation.MaxDeviationPercent / 100.0
	if maxDeviation == 0.0 {
		d.logger.Sugar().Errorf("maxDeviation=0 for continuous value analysis")
		return nil, fmt.Errorf("maxDeviation=0 for continuous value analysis")
	}

	for podName, deviation := range deviationByPods {
		_, found := podWithNoTraffic[podName]
		if found {
			d.logger.Sugar().Infof("the pod %s metrics are ignored now has it is marked out of traffic\n", podName)
			continue
		}

		if math.Abs(1-deviation) > maxDeviation {
			if p, ok := podByName[podName]; ok {
				// Only keeping known pod with too hig deviation
				result = append(result, p)
			}
		}
	}
	return result, nil
}
