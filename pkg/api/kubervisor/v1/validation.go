package v1

import (
	"fmt"
)

//ValidateBreakerStrategy validation of input
func ValidateBreakerStrategy(s BreakerStrategy) error {

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
		return fmt.Errorf("BreakerStrategy is missing anomaly detection specification (DiscreteValueOutOfList or CustomService)")
	}

	if len(strategies) > 1 {
		return fmt.Errorf("BreakerStrategy is defining multiple anomalies")
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
