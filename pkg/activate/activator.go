package activate

import (
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"
	kapiv1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	kv1 "k8s.io/client-go/listers/core/v1"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

//Activator engine that check anomaly and relabel pods
type Activator interface {
	Run(stop <-chan struct{})
	CompareConfig(specStrategy *api.ActivatorStrategy, specSelector labels.Selector) bool
}

//Config configuration required to create a Activator
type Config struct {
	KubervisorName          string
	BreakerStrategyName     string
	Selector                labels.Selector
	ActivatorStrategyConfig api.ActivatorStrategy

	PodLister  kv1.PodNamespaceLister
	PodControl pod.ControlInterface

	Logger *zap.Logger
}

var _ Activator = &ActivatorImpl{}

//ActivatorImpl implementation of the Activator interface
type ActivatorImpl struct {
	kubervisorName          string
	breakerStrategyName     string
	selector                labels.Selector
	activatorStrategyConfig api.ActivatorStrategy

	podLister  kv1.PodNamespaceLister
	podControl pod.ControlInterface

	logger *zap.Logger

	evaluationPeriod time.Duration
	strategyApplier  strategyApplier
}

type strategyApplier interface {
	applyActivatorStrategy(p *kapiv1.Pod) error
}

//Run implements Activator run loop ( to launch as goroutine: go Run())}
func (b *ActivatorImpl) Run(stop <-chan struct{}) {
	rqTrafficNo, err := labels.NewRequirement(labeling.LabelTrafficKey, selection.Equals, []string{string(labeling.LabelTrafficNo)})
	if err != nil {
		b.logger.Sugar().Errorf("unable to create labels.Requirement, error:%s", err)
		return
	}
	withTrafficNoSelector := b.selector.Add(*rqTrafficNo)

	ticker := time.NewTicker(b.evaluationPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			//Select pods affected by the associated breaker
			pods, err := b.podLister.List(withTrafficNoSelector)
			if err != nil {
				b.logger.Sugar().Errorf("activator for '%s' can't list pods:%s", b.kubervisorName, err)
			}

			for _, p := range pods {
				if err = b.strategyApplier.applyActivatorStrategy(p); err != nil {
					b.logger.Sugar().Errorf("can't apply activator '%s' strategy on pod '%s' :%v", b.kubervisorName, p.Name, err)
					continue
				}
			}
		case <-stop:
			return
		}
	}
}

// CompareConfig used to compare the current config with the possible new spec
func (b *ActivatorImpl) CompareConfig(specStrategy *api.ActivatorStrategy, specSelector labels.Selector) bool {
	if !apiequality.Semantic.DeepEqual(&b.activatorStrategyConfig, specStrategy) {
		return false
	}
	s, err := labeling.SelectorWithBreakerName(specSelector, b.kubervisorName)
	if err != nil {
		b.logger.Sugar().Errorf("unable to create selector from kubervisorName, error:%s", err)
	}
	return reflect.DeepEqual(s, b.selector)
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

	if b.activatorStrategyConfig.Period == nil {
		return fmt.Errorf("b.activatorStrategyConfig.Period is nil")
	}
	retryPeriod := time.Duration(*b.activatorStrategyConfig.Period*1000) * time.Millisecond
	now := time.Now()

	switch b.activatorStrategyConfig.Mode {
	case api.ActivatorStrategyModePeriodic:
		retrytime := breakAt.Add(retryPeriod)
		if retrytime.Before(now) {
			if _, err := b.podControl.UpdateActivationLabelsAndAnnotations(b.kubervisorName, p); err != nil {
				return err
			}
		}
	case api.ActivatorStrategyModeRetryAndKill:
		if retryCount > int(*b.activatorStrategyConfig.MaxRetryCount) {
			return b.podControl.KillPod(b.kubervisorName, p)
		}

		retrytime := breakAt.Add(time.Duration(retryCount) * retryPeriod)
		if retrytime.Before(now) {
			if _, err := b.podControl.UpdateActivationLabelsAndAnnotations(b.kubervisorName, p); err != nil {
				return err
			}
		}
	case api.ActivatorStrategyModeRetryAndPause:
		if retryCount > int(*b.activatorStrategyConfig.MaxRetryCount) {
			rqTrafficPause, err := labels.NewRequirement(labeling.LabelTrafficKey, selection.Equals, []string{string(labeling.LabelTrafficPause)})
			if err != nil {
				return fmt.Errorf("unable to create labels.Requirement, error:%s", err)
			}
			withTrafficPauseSelector := b.selector.Add(*rqTrafficPause)
			list, err := b.podLister.List(withTrafficPauseSelector)
			if err != nil {
				return fmt.Errorf("in activator '%s', can't list paused pods:%s", b.kubervisorName, err)
			}
			if len(list) >= int(*b.activatorStrategyConfig.MaxPauseCount) {
				return b.podControl.KillPod(b.kubervisorName, p)
			}
			if _, err := b.podControl.UpdatePauseLabelsAndAnnotations(b.kubervisorName, p); err != nil {
				return fmt.Errorf("in activator '%s' can't set 'pause' on pod '%s' :%s", b.kubervisorName, p.Name, err)
			}
			return nil
		}

		retrytime := breakAt.Add(time.Duration(retryCount) * retryPeriod)
		if retrytime.Before(now) {
			if _, err := b.podControl.UpdateActivationLabelsAndAnnotations(b.kubervisorName, p); err != nil {
				return fmt.Errorf("can't apply activator '%s' strategy on pod '%s' :%s", b.kubervisorName, p.Name, err)
			}
		}
	}
	return nil
}
