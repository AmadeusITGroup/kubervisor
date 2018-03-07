package activate

import (
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
)

//FactoryConfig parameters required for the creation of a Activator
type FactoryConfig struct {
	Config
	customFactory Factory
}

//Factory func for Activator
type Factory func(cfg FactoryConfig) (Activator, error)

var _ Factory = New

//New Factory for AnomalyDetection
func New(cfg FactoryConfig) (Activator, error) {
	if cfg.customFactory != nil {
		return cfg.customFactory(cfg)
	}

	augmentedSelector := labels.Everything()
	if cfg.Selector != nil {
		augmentedSelector = cfg.Selector.DeepCopySelector()
	}

	var err error
	rqBreaker, err := labels.NewRequirement(labeling.LabelBreakerNameKey, selection.Equals, []string{cfg.BreakerName})
	if err != nil {
		return nil, err
	}
	augmentedSelector = augmentedSelector.Add(*rqBreaker)

	a := &ActivatorImpl{
		activatorStrategyConfig: cfg.ActivatorStrategyConfig,
		logger:                  cfg.Logger,
		podControl:              cfg.PodControl,
		podLister:               cfg.PodLister,
		selectorConfig:          cfg.Selector,
		augmentedSelector:       augmentedSelector,
		breakerName:             cfg.BreakerName,
		evaluationPeriod:        time.Second,
	}
	a.strategyApplier = a
	return a, nil
}
