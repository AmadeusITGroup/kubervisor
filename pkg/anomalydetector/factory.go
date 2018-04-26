package anomalydetector

import (
	"fmt"

	promClient "github.com/prometheus/client_golang/api"

	"github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
)

//FactoryConfig parameters extended with factory features
type FactoryConfig struct {
	Config
	customFactory Factory
}

//Factory functor for AnomalyDetection
type Factory func(cfg FactoryConfig) (AnomalyDetector, error)

var _ Factory = New

//New Factory for AnomalyDetection
func New(cfg FactoryConfig) (AnomalyDetector, error) {

	switch {
	case cfg.BreakerStrategyConfig.DiscreteValueOutOfList != nil:
		return newDiscreteValueOutOfListAnalyser(cfg.Config)
	case cfg.BreakerStrategyConfig.ContinuousValueDeviation != nil:
		return newContinuousValueDeviation(cfg.Config)
	case cfg.BreakerStrategyConfig.CustomService != "":
		return newCustomAnalyser(cfg.Config)
	case cfg.customFactory != nil:
		return cfg.customFactory(cfg)
	default:
		return nil, fmt.Errorf("no anomaly detection could be built, missing definition")
	}
}

func newCustomAnalyser(cfg Config) (*CustomAnomalyDetector, error) {
	c := &CustomAnomalyDetector{
		serviceURI: cfg.BreakerStrategyConfig.CustomService,
		selector:   cfg.Selector,
		logger:     cfg.Logger,
	}
	c.init()
	return c, nil
}

func newDiscreteValueOutOfListAnalyser(cfg Config) (*DiscreteValueOutOfListAnalyser, error) {
	analyserCfg := *cfg.BreakerStrategyConfig.DiscreteValueOutOfList

	if err := v1.ValidateDiscreteValueOutOfList(analyserCfg); err != nil {
		return nil, err
	}

	good, bad := analyserCfg.GoodValues, analyserCfg.BadValues
	valueCheckerFunc := func(value string) bool { return ContainsString(good, value) }
	if len(good) == 0 && len(bad) != 0 {
		valueCheckerFunc = func(value string) bool { return !ContainsString(bad, value) }
	}

	a := &DiscreteValueOutOfListAnalyser{DiscreteValueOutOfList: analyserCfg, selector: cfg.Selector, podLister: cfg.PodLister, logger: cfg.Logger}
	switch {
	case analyserCfg.PromQL != "":

		analyser := &promDiscreteValueOutOfListAnalyser{config: analyserCfg, logger: cfg.Logger}

		analyser.valueCheckerFunc = valueCheckerFunc
		promconfig := promClient.Config{Address: "http://" + analyserCfg.PrometheusService}
		var err error
		if analyser.prometheusClient, err = promClient.NewClient(promconfig); err != nil {
			return nil, err
		}
		a.analyser = analyser
	default:
		return nil, fmt.Errorf("missing parameter to create DiscreteValueOutOfListAnalyser")
	}
	return a, nil
}

func newContinuousValueDeviation(cfg Config) (*ContinuousValueDeviationAnalyser, error) {
	analyserCfg := *cfg.BreakerStrategyConfig.ContinuousValueDeviation

	if err := v1.ValidateContinuousValueDeviation(analyserCfg); err != nil {
		return nil, err
	}
	a := &ContinuousValueDeviationAnalyser{ContinuousValueDeviation: analyserCfg, selector: cfg.Selector, podLister: cfg.PodLister, logger: cfg.Logger}
	switch {
	case analyserCfg.PromQL != "":

		analyser := &promContinuousValueDeviationAnalyser{config: analyserCfg, logger: cfg.Logger}
		promconfig := promClient.Config{Address: "http://" + analyserCfg.PrometheusService}
		var err error
		if analyser.prometheusClient, err = promClient.NewClient(promconfig); err != nil {
			return nil, err
		}
		a.analyser = analyser
	default:
		return nil, fmt.Errorf("missing parameter to create ContinuousValueDeviationAnalyser")
	}
	return a, nil

}

// ContainsString checks if the slice has the contains value in it.
func ContainsString(slice []string, contains string) bool {
	for _, value := range slice {
		if value == contains {
			return true
		}
	}
	return false
}
