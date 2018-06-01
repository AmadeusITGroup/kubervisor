package breaker

import (
	"fmt"

	"github.com/amadeusitgroup/kubervisor/pkg/anomalydetector"
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

	anomalyDetector, err := anomalydetector.New(anomalydetector.FactoryConfig{
		Config: anomalydetector.Config{
			BreakerStrategyConfig: cfg.BreakerStrategyConfig,
			Selector:              cfg.Selector,
			Logger:                cfg.Logger,
			PodLister:             cfg.PodLister,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("can't create breaker: %s", err)
	}

	return &breakerImpl{
		breakerStrategyName:   cfg.StrategyName,
		breakerStrategyConfig: cfg.BreakerStrategyConfig,
		logger:                cfg.Logger,
		podControl:            cfg.PodControl,
		podLister:             cfg.PodLister,
		kubervisorName:        cfg.KubervisorName,
		selector:              cfg.Selector,
		anomalyDetector:       anomalyDetector,
	}, nil
}
