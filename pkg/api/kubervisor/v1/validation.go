package v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/validation"
)

//ValidateKubervisorServiceSpec validate the KubervisorService specification
func ValidateKubervisorServiceSpec(s KubervisorServiceSpec) error {
	valStr := validation.NameIsDNS1035Label(s.Service, false)
	if len(valStr) != 0 {
		return fmt.Errorf("Validation of kubervisor service specification, service: %s", valStr[0])
	}

	if s.Breakers == nil || len(s.Breakers) == 0 {
		return fmt.Errorf("Validation of kubervisor service specification failed: no BreakerStrategy defined")
	}
	names := map[string]struct{}{}
	for i := range s.Breakers {
		if err := ValidateBreakerStrategy(s.Breakers[i]); err != nil {
			return fmt.Errorf("Validation of kubervisor service specification failed for breaker strategy %d: %v", i, err)
		}
		name := s.Breakers[i].Name
		if _, ok := names[name]; ok {
			return fmt.Errorf("Validation of kubervisor service specification: breaker strategy name not unique %d: %s", i, name)
		}
		names[name] = struct{}{}
	}

	if err := ValidateActivatorStrategy(s.DefaultActivator); err != nil {
		return fmt.Errorf("Validation of kubervisor service specification failed for default activator strategy: %v", err)
	}
	return nil
}

//ValidateActivatorStrategy validation of input
func ValidateActivatorStrategy(s ActivatorStrategy) error {
	return nil // TODO #20
}

//ValidateBreakerStrategy validation of input
func ValidateBreakerStrategy(s BreakerStrategy) error {
	valStr := validation.NameIsDNS1035Label(s.Name, false)
	if len(valStr) != 0 {
		return fmt.Errorf("bad strategy name: %s", valStr[0])
	}

	strategies := []string{}
	if s.DiscreteValueOutOfList != nil {
		strategies = append(strategies, "DiscreteValueOutOfList")
		if err := ValidateDiscreteValueOutOfList(*s.DiscreteValueOutOfList); err != nil {
			return fmt.Errorf("Validation of strategy DiscreteValueOutOfList failed: %v", err)
		}
	}
	if s.ContinuousValueDeviation != nil {
		strategies = append(strategies, "ContinuousValueDeviation")
		if err := ValidateContinuousValueDeviation(*s.ContinuousValueDeviation); err != nil {
			return fmt.Errorf("Validation of strategy ContinuousValueDeviation failed: %v", err)
		}
	}
	if s.CustomService != "" {
		strategies = append(strategies, "CustomService")
	}

	if len(strategies) == 0 {
		return fmt.Errorf("BreakerStrategy is missing anomaly detection specification (DiscreteValueOutOfList or CustomService or ...)")
	}

	if len(strategies) > 1 {
		return fmt.Errorf("BreakerStrategy is defining multiple anomalies")
	}

	if s.EvaluationPeriod != nil && *s.EvaluationPeriod <= 0.01 {
		return fmt.Errorf("BreakerStrategy evaluation period undefined or too small (less than 10 ms)")
	}

	if s.EvaluationPeriod != nil && *s.EvaluationPeriod > 24*3600.0 {
		return fmt.Errorf("BreakerStrategy evaluation period undefined or too big (more than 1 day)")
	}

	if s.Activator != nil {
		if err := ValidateActivatorStrategy(*s.Activator); err != nil {
			return fmt.Errorf("BreakerStrategy activator is invalid: %v", err)
		}
	}

	return nil
}

//ValidateDiscreteValueOutOfList validation of input
func ValidateDiscreteValueOutOfList(d DiscreteValueOutOfList) error {
	good, bad := d.GoodValues, d.BadValues
	if len(good) == 0 && len(bad) == 0 {
		return fmt.Errorf("no good nor bad value defined")
	}
	if len(good) != 0 && len(bad) != 0 {
		return fmt.Errorf("good and bad value defined, only good values will be used to do inclusion")
	}

	if len(d.Key) == 0 {
		return fmt.Errorf("missing metric Key definition")
	}

	if len(d.PodNameKey) == 0 {
		return fmt.Errorf("missing PodName Key definition")
	}

	switch {
	case d.PromQL != "" || d.PrometheusService != "":
		if d.PrometheusService == "" {
			return fmt.Errorf("missing Prometheus service")
		}
		if d.PromQL == "" {
			return fmt.Errorf("missing PromQL")
		}
	default:
		return fmt.Errorf("missing parameter to create DiscreteValueOutOfListAnalyser")
	}
	return nil
}

//ValidateContinuousValueDeviation validation of input
func ValidateContinuousValueDeviation(d ContinuousValueDeviation) error {
	if len(d.PodNameKey) == 0 {
		return fmt.Errorf("missing PodName Key definition")
	}

	if d.MaxDeviationPercent == nil {
		return fmt.Errorf("missing Max Deviation percent")
	}

	switch {
	case d.PromQL != "" || d.PrometheusService != "":
		if d.PrometheusService == "" {
			return fmt.Errorf("missing Prometheus service")
		}
		if d.PromQL == "" {
			return fmt.Errorf("missing PromQL")
		}
	default:
		return fmt.Errorf("missing parameter to create DiscreteValueOutOfListAnalyser")
	}
	return nil
}
