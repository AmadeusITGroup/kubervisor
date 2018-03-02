package breaker

import (
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
}

//Config configuration required to create a Breaker
type Config struct {
	BreakerStrategyConfig v1.BreakerStrategy
	Selector              labels.Selector
	PodLister             kv1.PodLister
	PodControl            pod.ControlInterface
	Logger                *zap.Logger
}

var _ Breaker = &BreakerImpl{}

//BreakerImpl implementation of the breaker interface
type BreakerImpl struct {
	breakerStrategyConfig v1.BreakerStrategy
	selector              labels.Selector
	podLister             kv1.PodLister
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
			b.logger.Sugar().Infof("All %d", len(allPods))
			runningPods := pod.KeepRunningPods(allPods)
			b.logger.Sugar().Infof("runningPods %d", len(runningPods))
			readyPods := pod.PurgeNotReadyPods(runningPods)
			b.logger.Sugar().Infof("readyPods %d", len(readyPods))
			withTraffic := pod.KeepWithTrafficYesPods(readyPods)
			b.logger.Sugar().Infof("withTraffic %d", len(withTraffic))
			removeCount := len(withTraffic) - b.computeMinAvailablePods(len(withTraffic))
			b.logger.Sugar().Infof("RemoveCompute %d", removeCount)
			if removeCount > len(podsToCut) {
				removeCount = len(podsToCut)
			}
			if removeCount < 0 {
				removeCount = 0
			}

			b.logger.Sugar().Infof("RemoveAdjust %d", removeCount)

			for _, p := range podsToCut[:removeCount] {
				b.podControl.UpdateBreakerAnnotationAndLabel(p)
			}

		case <-stop:
			return
		}
	}
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
