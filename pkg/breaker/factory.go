package breaker

import (
	"fmt"

	"github.com/amadeusitgroup/podkubervisor/pkg/anomalydetector"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
)

//FactoryConfig parameters required for the creation of a breaker
type FactoryConfig struct {
	Config
	customFactory Factory
}

//Factory func for Breaker
type Factory func(cfg FactoryConfig) (Breaker, error)

var _ Factory = New

//New Factory for AnomalyDetection
func New(cfg FactoryConfig) (Breaker, error) {
	if cfg.customFactory != nil {
		return cfg.customFactory(cfg)
	}

	augmentedSelector, errSelector := labeling.SelectorWithBreakerName(cfg.Selector, cfg.BreakerName)
	if errSelector != nil {
		return nil, fmt.Errorf("Can't build breaker: %v", errSelector)
	}

	anomalyDetector, err := anomalydetector.New(anomalydetector.FactoryConfig{
		Config: anomalydetector.Config{
			BreakerStrategyConfig: cfg.BreakerStrategyConfig,
			Selector:              augmentedSelector,
			Logger:                cfg.Logger,
			PodLister:             cfg.PodLister,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("can't create breaker: %s", err)
	}

	return &BreakerImpl{
		KubervisorServiceName: cfg.KubervisorServiceName,
		breakerStrategyConfig: cfg.BreakerStrategyConfig,
		logger:                cfg.Logger,
		podControl:            cfg.PodControl,
		podLister:             cfg.PodLister,
		selector:              augmentedSelector,
		anomalyDetector:       anomalyDetector,
	}, nil
}
