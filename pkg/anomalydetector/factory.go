package anomalydetector

import (
	"fmt"

	promClient "github.com/prometheus/client_golang/api"
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
	case cfg.customFactory != nil:
		return cfg.customFactory(cfg)
	default:
		return nil, fmt.Errorf("no anomaly detection could be built, missing definition")
	}
}

func newDiscreteValueOutOfListAnalyser(cfg Config) (*DiscreteValueOutOfListAnalyser, error) {
	analyserCfg := *cfg.BreakerStrategyConfig.DiscreteValueOutOfList

	good, bad := analyserCfg.GoodValues, analyserCfg.BadValues
	if len(good) == 0 && len(bad) == 0 {
		return nil, fmt.Errorf("no good nor bad value defined")
	}
	if len(good) != 0 && len(bad) != 0 {
		return nil, fmt.Errorf("good and bad value defined, only good values will be used to do inclusion")
	}
	valueCheckerFunc := func(value string) bool { return ContainsString(good, value) }
	if len(good) == 0 && len(bad) != 0 {
		valueCheckerFunc = func(value string) bool { return !ContainsString(bad, value) }
	}

	if len(analyserCfg.Key) == 0 {
		return nil, fmt.Errorf("missing metric Key definition")
	}

	if len(analyserCfg.PodNameKey) == 0 {
		return nil, fmt.Errorf("missing PodName Key definition")
	}

	a := &DiscreteValueOutOfListAnalyser{DiscreteValueOutOfList: analyserCfg, selector: cfg.Selector, podLister: cfg.PodLister, logger: cfg.Logger}
	switch {
	case analyserCfg.PromQL != "":

		podAnalyser := &promDiscreteValueOutOfListAnalyser{config: analyserCfg, logger: cfg.Logger}

		podAnalyser.valueCheckerFunc = valueCheckerFunc

		if analyserCfg.PrometheusService == "" {
			return nil, fmt.Errorf("missing Prometheus service")
		}

		promconfig := promClient.Config{Address: "http://" + analyserCfg.PrometheusService}
		var err error
		if podAnalyser.prometheusClient, err = promClient.NewClient(promconfig); err != nil {
			return nil, err
		}

		a.podAnalyser = podAnalyser
	default:
		return nil, fmt.Errorf("missing parameter to create DiscreteValueOutOfListAnalyser")
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
