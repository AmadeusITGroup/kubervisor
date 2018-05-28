package activate

import (
	"fmt"
	"time"

	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
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

	augmentedSelector, errSelector := labeling.SelectorWithBreakerName(cfg.Selector, cfg.BreakerName)
	if errSelector != nil {
		return nil, fmt.Errorf("Can't build activator: %v", errSelector)
	}

	a := &ActivatorImpl{
		activatorStrategyConfig: cfg.ActivatorStrategyConfig,
		logger:                  cfg.Logger,
		podControl:              cfg.PodControl,
		podLister:               cfg.PodLister,
		selectorConfig:          augmentedSelector,
		breakerName:             cfg.BreakerName,
		breakerStrategyName:     cfg.BreakerStrategyName,
		evaluationPeriod:        time.Second,
	}
	a.strategyApplier = a
	return a, nil
}
