package breaker

import (
	"reflect"
	"time"

	"go.uber.org/zap"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/kubervisor/pkg/anomalydetector"
	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

//Breaker engine that check anomaly and relabel pods
type Breaker interface {
	Run(stop <-chan struct{})
	CompareConfig(specConfig *api.BreakerStrategy, specSelector labels.Selector) bool
	Name() string
}

//Config configuration required to create a Breaker
type Config struct {
	KubervisorName        string
	StrategyName          string
	Selector              labels.Selector
	BreakerStrategyConfig api.BreakerStrategy

	PodLister  kv1.PodNamespaceLister
	PodControl pod.ControlInterface

	Logger *zap.Logger
}

var _ Breaker = &breakerImpl{}

//breakerImpl implementation of the breaker interface
type breakerImpl struct {
	kubervisorName        string
	breakerStrategyName   string
	selector              labels.Selector
	breakerStrategyConfig api.BreakerStrategy

	podLister  kv1.PodNamespaceLister
	podControl pod.ControlInterface

	logger *zap.Logger

	anomalyDetector anomalydetector.AnomalyDetector
}

//Name return the name of the breaker strategy
func (b *breakerImpl) Name() string {
	return b.breakerStrategyName
}

//Run implements Breaker run loop ( to launch as goroutine: go Run())
func (b *breakerImpl) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Duration(*b.breakerStrategyConfig.EvaluationPeriod*1000) * time.Millisecond)
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

			allPods, err := b.podLister.List(b.selector)
			if err != nil {
				b.logger.Sugar().Errorf("can't list pods, error: %v ", err)
				continue
			}
			runningPods, err := pod.KeepRunningPods(allPods)
			if err != nil {
				b.logger.Sugar().Errorf("can't get running pods, error: %v ", err)
				continue
			}
			readyPods, err := pod.PurgeNotReadyPods(runningPods)
			if err != nil {
				b.logger.Sugar().Errorf("can't purge not ready pods, error: %v ", err)
				continue
			}
			withTraffic, err := pod.KeepWithTrafficYesPods(readyPods)
			if err != nil {
				b.logger.Sugar().Errorf("can't get pods with traffic, error: %v ", err)
				continue
			}
			removeCount := len(withTraffic) - b.computeMinAvailablePods(len(withTraffic))

			if removeCount > len(podsToCut) {
				removeCount = len(podsToCut)
			}
			if removeCount < 0 {
				removeCount = 0
			}

			for _, p := range podsToCut[:removeCount] {
				if _, err := b.podControl.UpdateBreakerAnnotationAndLabel(b.kubervisorName, b.breakerStrategyName, p); err != nil {
					b.logger.Sugar().Errorf("can't update Breaker annotation and label: %s", err)
				}
			}

		case <-stop:
			return
		}
	}
}

// CompareConfig used to compare the current config with a possible new spec config
func (b *breakerImpl) CompareConfig(specConfig *api.BreakerStrategy, specSelector labels.Selector) bool {
	if !apiequality.Semantic.DeepEqual(&b.breakerStrategyConfig, specConfig) {
		return false
	}
	s, err := labeling.SelectorWithBreakerName(specSelector, b.kubervisorName)
	if err != nil {
		b.logger.Sugar().Errorf("unable to create Selector with breaker name %s, error:%v", b.kubervisorName, err)
	}
	return reflect.DeepEqual(s, b.selector)

}

func (b *breakerImpl) computeMinAvailablePods(podUnderSelectorCount int) int {
	count, ratio := 0, 0
	if b.breakerStrategyConfig.MinPodsAvailableRatio != nil {
		ratio = int(*b.breakerStrategyConfig.MinPodsAvailableRatio)
	}
	if b.breakerStrategyConfig.MinPodsAvailableCount != nil {
		count = int(*b.breakerStrategyConfig.MinPodsAvailableCount)
	}
	quota := podUnderSelectorCount * ratio / 100
	if quota > count {
		return quota
	}
	return count
}
