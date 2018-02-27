package anomalyDetector

import (
	"fmt"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"go.uber.org/zap"
	kv1 "k8s.io/client-go/listers/core/v1"

	promClient "github.com/prometheus/client_golang/api"
)

//Config parameters required for the creation of an AnomalyDetector
type Config struct {
	BreakerConfig v1.BreakerConfigSpec
	PodLister     kv1.PodLister
	Logger        *zap.Logger
	customFactory Factory
}

//Factory functor for AnomalyDetection
type Factory func(cfg Config) (AnomalyDetector, error)

var _ Factory = New

//New Factory for AnomalyDetection
func New(cfg Config) (AnomalyDetector, error) {

	switch {
	case cfg.BreakerConfig.Breaker.DiscreteValueOutOfList != nil:
		{
			return newDiscreteValueOutOfListAnalyser(cfg)
		}
	case cfg.customFactory != nil:
		return cfg.customFactory(cfg)
	default:
		return nil, fmt.Errorf("no anomaly detection could be built, missing definition")
	}
}

func newDiscreteValueOutOfListAnalyser(cfg Config) (*DiscreteValueOutOfListAnalyser, error) {
	a := &DiscreteValueOutOfListAnalyser{BreakerConfigSpec: cfg.BreakerConfig, podLister: cfg.PodLister, logger: cfg.Logger}
	switch {
	case cfg.BreakerConfig.Breaker.DiscreteValueOutOfList.PromQL != "":

		podAnalyser := &promDiscreteValueOutOfListAnalyser{config: *cfg.BreakerConfig.Breaker.DiscreteValueOutOfList, logger: cfg.Logger}
		if cfg.BreakerConfig.Breaker.DiscreteValueOutOfList.PrometheusService == "" {
			return nil, fmt.Errorf("missing Prometheus service")
		}

		promconfig := promClient.Config{Address: "http://" + cfg.BreakerConfig.Breaker.DiscreteValueOutOfList.PrometheusService}
		var err error
		if podAnalyser.prometheusClient, err = promClient.NewClient(promconfig); err != nil {
			return nil, err
		}

		good, bad := cfg.BreakerConfig.Breaker.DiscreteValueOutOfList.GoodValues, cfg.BreakerConfig.Breaker.DiscreteValueOutOfList.BadValues
		if len(good) == 0 && len(bad) == 0 {
			return nil, fmt.Errorf("no good nor bad value defined")
		}
		if len(good) != 0 && len(bad) != 0 {
			return nil, fmt.Errorf("good and bad value defined, only good values will be used to do inclusion")
		}
		podAnalyser.valueCheckerFunc = func(value string) bool { return ContainsString(good, value) }
		if len(good) == 0 && len(bad) != 0 {
			podAnalyser.valueCheckerFunc = func(value string) bool { return !ContainsString(bad, value) }
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
