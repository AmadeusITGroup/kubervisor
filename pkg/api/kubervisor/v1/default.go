package v1

// DefaultKubervisorService injecting default values for the struct
func DefaultKubervisorService(item *KubervisorService) *KubervisorService {
	copy := item.DeepCopy()
	copy.Spec.Activator = *DefaultActivatorStrategy(&copy.Spec.Activator)
	copy.Spec.Breaker = *DefaultBreakerStrategy(&copy.Spec.Breaker)

	return copy
}

//DefaultDiscreteValueOutOfList injecting default values for the struct
func DefaultDiscreteValueOutOfList(item *DiscreteValueOutOfList) *DiscreteValueOutOfList {
	copy := item.DeepCopy()
	if copy.BadValues == nil {
		copy.BadValues = []string{}
	}
	if copy.GoodValues == nil {
		copy.GoodValues = []string{}
	}
	if copy.MinimumActivityCount == nil {
		copy.MinimumActivityCount = NewUInt(2)
	}
	if copy.TolerancePercent == nil {
		copy.TolerancePercent = NewUInt(50)
	}
	return copy
}

//DefaultContinuousValueDeviation injecting default values for the struct
func DefaultContinuousValueDeviation(item *ContinuousValueDeviation) *ContinuousValueDeviation {
	copy := item.DeepCopy()
	if copy.MaxDeviationPercent == nil {
		copy.MaxDeviationPercent = NewFloat64(75)
	}
	return copy
}

// DefaultBreakerStrategy injecting default values for the struct
func DefaultBreakerStrategy(item *BreakerStrategy) *BreakerStrategy {
	copy := item.DeepCopy()
	if copy.EvaluationPeriod == nil {
		copy.EvaluationPeriod = NewFloat64(5)
	}
	if copy.MinPodsAvailableCount == nil {
		copy.MinPodsAvailableCount = NewUInt(1)
	}
	if copy.DiscreteValueOutOfList != nil {
		copy.DiscreteValueOutOfList = DefaultDiscreteValueOutOfList(copy.DiscreteValueOutOfList)
	}
	if copy.ContinuousValueDeviation != nil {
		copy.ContinuousValueDeviation = DefaultContinuousValueDeviation(copy.ContinuousValueDeviation)
	}

	return copy
}

//DefaultActivatorStrategy injecting default values for the struct
func DefaultActivatorStrategy(item *ActivatorStrategy) *ActivatorStrategy {
	copy := item.DeepCopy()
	if copy.MaxPauseCount == nil {
		copy.MaxPauseCount = NewUInt(1)
	}
	if copy.MaxRetryCount == nil {
		copy.MaxRetryCount = NewUInt(3)
	}
	if copy.Mode == "" {
		copy.Mode = ActivatorStrategyModePeriodic
	}
	if copy.Period == nil {
		copy.Period = NewFloat64(10.0)
	}
	return copy
}

// NewUInt return a pointer to a uint
func NewUInt(val uint) *uint {
	output := new(uint)
	*output = val
	return output
}

// NewFloat64 return a pointer to a float64
func NewFloat64(val float64) *float64 {
	output := new(float64)
	*output = val
	return output
}

// IsKubervisorServiceDefaulted used to check if a KubervisorService is already defaulted
func IsKubervisorServiceDefaulted(bc *KubervisorService) bool {
	if !isActivatorStrategyDefaulted(&bc.Spec.Activator) {
		return false
	}
	if !isBreakerStrategyDefaulted(&bc.Spec.Breaker) {
		return false
	}
	return true
}

// isActivatorStrategyDefaulted used to check if a ActivatorStrategy is already defaulted
func isActivatorStrategyDefaulted(item *ActivatorStrategy) bool {
	if item.MaxPauseCount == nil {
		return false
	}
	if item.MaxRetryCount == nil {
		return false
	}
	if item.Mode == "" {
		return false
	}
	if item.Period == nil {
		return false
	}
	return true
}

// isBreakerStrategyDefaulted injecting default values for the struct
func isBreakerStrategyDefaulted(item *BreakerStrategy) bool {
	if item.EvaluationPeriod == nil {
		return false
	}
	if item.MinPodsAvailableCount == nil && item.MinPodsAvailableRatio == nil {
		return false
	}
	if item.DiscreteValueOutOfList != nil {
		if !isDiscreteValueOutOfListDefaulted(item.DiscreteValueOutOfList) {
			return false
		}
	}
	if item.ContinuousValueDeviation != nil {
		return isContinuousValueDeviationDefaulted(item.ContinuousValueDeviation)
	}
	return true
}

// isDiscreteValueOutOfListDefaulted used to check if a DiscreteValueOutOfList is already defaulted
func isDiscreteValueOutOfListDefaulted(item *DiscreteValueOutOfList) bool {
	if item.MinimumActivityCount == nil {
		return false
	}
	if item.TolerancePercent == nil {
		return false
	}
	return true
}

// isDiscreteValueOutOfListDefaulted used to check if a DiscreteValueOutOfList is already defaulted
func isContinuousValueDeviationDefaulted(item *ContinuousValueDeviation) bool {
	if item.MaxDeviationPercent == nil {
		return false
	}
	return true
}
