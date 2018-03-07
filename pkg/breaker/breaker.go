package breaker

import (
	"reflect"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/anomalydetector"
	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/pod"
)

//Breaker engine that check anomaly and relabel pods
type Breaker interface {
	Run(stop <-chan struct{})
	CompareConfig(specConfig *v1.BreakerStrategy) bool
}

//Config configuration required to create a Breaker
type Config struct {
	BreakerStrategyConfig v1.BreakerStrategy
	BreakerConfigName     string
	Selector              labels.Selector
	PodLister             kv1.PodNamespaceLister
	PodControl            pod.ControlInterface
	Logger                *zap.Logger
}

var _ Breaker = &BreakerImpl{}

//BreakerImpl implementation of the breaker interface
type BreakerImpl struct {
	BreakerConfigName     string
	breakerStrategyConfig v1.BreakerStrategy
	selector              labels.Selector
	podLister             kv1.PodNamespaceLister
	podControl            pod.ControlInterface

	logger          *zap.Logger
	anomalyDetector anomalydetector.AnomalyDetector
}

//Run implements Breaker run loop ( to launch as goroutine: go Run())
func (b *BreakerImpl) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(b.breakerStrategyConfig.EvaluationPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			podsToCut, err := b.anomalyDetector.GetPodsOutOfBounds()
			if err != nil {
				b.logger.Sugar().Errorf("can't apply breaker. Anomaly detection failed: %s", err)
				continue
			}

			if len(podsToCut) == 0 {
				b.logger.Sugar().Debug("no anomaly detected.")
				continue
			}

			allPods, _ := b.podLister.List(b.selector)
			runningPods := pod.KeepRunningPods(allPods)
			readyPods := pod.PurgeNotReadyPods(runningPods)
			withTraffic := pod.KeepWithTrafficYesPods(readyPods)
			removeCount := len(withTraffic) - b.computeMinAvailablePods(len(withTraffic))

			if removeCount > len(podsToCut) {
				removeCount = len(podsToCut)
			}
			if removeCount < 0 {
				removeCount = 0
			}

			for _, p := range podsToCut[:removeCount] {
				b.podControl.UpdateBreakerAnnotationAndLabel(b.BreakerConfigName, p)
			}

		case <-stop:
			return
		}
	}
}

// CompareConfig used to compare the current config with a possible new spec config
func (b *BreakerImpl) CompareConfig(specConfig *v1.BreakerStrategy) bool {
	if !reflect.DeepEqual(&b.breakerStrategyConfig, specConfig) {
		return false
	}
	return true
}

func (b *BreakerImpl) computeMinAvailablePods(podUnderSelectorCount int) int {
	count, ratio := 0, 0
	if b.breakerStrategyConfig.MinPodsAvailableRatio != nil {
		ratio = int(*b.breakerStrategyConfig.MinPodsAvailableRatio)
	}
	if b.breakerStrategyConfig.MinPodsAvailableCount != nil {
		count = int(*b.breakerStrategyConfig.MinPodsAvailableCount)
	}
	quota := podUnderSelectorCount * int(ratio) / 100
	if quota > count {
		return quota
	}
	return count
}
