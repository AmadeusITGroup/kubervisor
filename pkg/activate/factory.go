package activate

import (
	"time"
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

	a := &ActivatorImpl{
		kubervisorName:          cfg.KubervisorName,
		breakerStrategyName:     cfg.BreakerStrategyName,
		selector:                cfg.Selector,
		activatorStrategyConfig: cfg.ActivatorStrategyConfig,
		logger:                  cfg.Logger,
		podControl:              cfg.PodControl,
		podLister:               cfg.PodLister,
		evaluationPeriod:        time.Second,
	}
	a.strategyApplier = a
	return a, nil
}
