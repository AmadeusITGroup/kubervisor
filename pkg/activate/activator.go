package activate

import (
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	kv1 "k8s.io/client-go/listers/core/v1"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
	"github.com/amadeusitgroup/podkubervisor/pkg/pod"
)

//Activator engine that check anomaly and relabel pods
type Activator interface {
	Run(stop <-chan struct{})
	CompareConfig(specStrategy *v1.ActivatorStrategy, specSelector labels.Selector) bool
}

//Config configuration required to create a Activator
type Config struct {
	ActivatorStrategyConfig v1.ActivatorStrategy
	Selector                labels.Selector
	PodLister               kv1.PodNamespaceLister
	PodControl              pod.ControlInterface
	BreakerName             string
	Logger                  *zap.Logger
}

var _ Activator = &ActivatorImpl{}

//ActivatorImpl implementation of the Activator interface
type ActivatorImpl struct {
	activatorStrategyConfig v1.ActivatorStrategy
	selectorConfig          labels.Selector
	augmentedSelector       labels.Selector
	podLister               kv1.PodNamespaceLister
	podControl              pod.ControlInterface
	breakerName             string
	logger                  *zap.Logger
	evaluationPeriod        time.Duration
	strategyApplier         strategyApplier
}

type strategyApplier interface {
	applyActivatorStrategy(p *kapiv1.Pod) error
}

//Run implements Activator run loop ( to launch as goroutine: go Run())}
func (b *ActivatorImpl) Run(stop <-chan struct{}) {
	rqTrafficNo, _ := labels.NewRequirement(labeling.LabelTrafficKey, selection.Equals, []string{string(labeling.LabelTrafficNo)})
	withTrafficNoSelector := b.augmentedSelector.Add(*rqTrafficNo)

	ticker := time.NewTicker(b.evaluationPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			//Select pods affected by the associated breaker
			pods, err := b.podLister.List(withTrafficNoSelector)
			if err != nil {
				b.logger.Sugar().Errorf("activator for '%s' can't list pods:%s", b.breakerName, err)
			}

			for _, p := range pods {
				if err = b.strategyApplier.applyActivatorStrategy(p); err != nil {
					b.logger.Sugar().Errorf("can't apply activator '%s' strategy on pod '%s' :%s", b.breakerName, p.Name, err)
				}
			}
		case <-stop:
			return
		}
	}
}

// CompareConfig used to compare the current config with the possible new spec
func (b *ActivatorImpl) CompareConfig(specStrategy *v1.ActivatorStrategy, specSelector labels.Selector) bool {
	if !reflect.DeepEqual(&b.activatorStrategyConfig, specStrategy) {
		return false
	}
	if !reflect.DeepEqual(b.selectorConfig, specSelector) {
		return false
	}

	return true
}

func (b *ActivatorImpl) applyActivatorStrategy(p *kapiv1.Pod) error {
	breakAt, err := labeling.GetBreakAt(p)
	if err != nil {
		return err
	}
	retryCount, err := labeling.GetRetryCount(p)
	if err != nil {
		return err
	}

	retryPeriod := b.activatorStrategyConfig.Period
	now := time.Now()

	switch b.activatorStrategyConfig.Mode {
	case v1.ActivatorStrategyModePeriodic:
		retrytime := breakAt.Add(retryPeriod)
		if retrytime.Before(now) {
			if _, err := b.podControl.UpdateActivationLabelsAndAnnotations(p); err != nil {
				return err
			}
		}
	case v1.ActivatorStrategyModeRetryAndKill:
		if retryCount > int(*b.activatorStrategyConfig.MaxRetryCount) {
			return b.podControl.KillPod(p)
		}

		retrytime := breakAt.Add(time.Duration(retryCount) * retryPeriod)
		if retrytime.Before(now) {
			if _, err := b.podControl.UpdateActivationLabelsAndAnnotations(p); err != nil {
				return err
			}
		}
	case v1.ActivatorStrategyModeRetryAndPause:
		if retryCount > int(*b.activatorStrategyConfig.MaxRetryCount) {
			rqTrafficPause, _ := labels.NewRequirement(labeling.LabelTrafficKey, selection.Equals, []string{string(labeling.LabelTrafficPause)})
			withTrafficPauseSelector := b.augmentedSelector.Add(*rqTrafficPause)
			list, err := b.podLister.List(withTrafficPauseSelector)
			if err != nil {
				return fmt.Errorf("in activator '%s', can't list paused pods:%s", b.breakerName, err)
			}
			if len(list) >= int(*b.activatorStrategyConfig.MaxPauseCount) {
				return b.podControl.KillPod(p)
			}
			if _, err := b.podControl.UpdatePauseLabelsAndAnnotations(p); err != nil {
				return fmt.Errorf("in activator '%s' can't set 'pause' on pod '%s' :%s", b.breakerName, p.Name, err)
			}
			return nil
		}

		retrytime := breakAt.Add(time.Duration(retryCount) * retryPeriod)
		if retrytime.Before(now) {
			if _, err := b.podControl.UpdateActivationLabelsAndAnnotations(p); err != nil {
				return fmt.Errorf("can't apply activator '%s' strategy on pod '%s' :%s", b.breakerName, p.Name, err)
			}
		}
	}
	return nil
}
