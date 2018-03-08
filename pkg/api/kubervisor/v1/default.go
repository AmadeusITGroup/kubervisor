package v1

import "time"

// DefaultBreakerConfig injecting default values for the struct
func DefaultBreakerConfig(item *BreakerConfig) *BreakerConfig {
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

// DefaultBreakerStrategy injecting default values for the struct
func DefaultBreakerStrategy(item *BreakerStrategy) *BreakerStrategy {
	copy := item.DeepCopy()
	if copy.EvaluationPeriod == time.Duration(0) {
		copy.EvaluationPeriod = time.Duration(5 * time.Second)
	}
	if copy.MinPodsAvailableCount == nil {
		copy.MinPodsAvailableCount = NewUInt(1)
	}
	if copy.MinPodsAvailableRatio == nil {
		copy.MinPodsAvailableRatio = NewUInt(70)
	}
	if copy.DiscreteValueOutOfList == nil {
		copy.DiscreteValueOutOfList = &DiscreteValueOutOfList{}
	}
	copy.DiscreteValueOutOfList = DefaultDiscreteValueOutOfList(copy.DiscreteValueOutOfList)

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
	if copy.Period == 0 {
		copy.Period = time.Duration(10)
	}
	return copy
}

// NewUInt return a pointer to a uint
func NewUInt(val uint) *uint {
	output := new(uint)
	*output = val
	return output
}

// IsBreakerConfigDefaulted used to check if a BreakerConfig is already defaulted
func IsBreakerConfigDefaulted(bc *BreakerConfig) bool {
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
	if item.Period == 0 {
		return false
	}
	return true
}

// isBreakerStrategyDefaulted injecting default values for the struct
func isBreakerStrategyDefaulted(item *BreakerStrategy) bool {
	if item.EvaluationPeriod == time.Duration(0) {
		return false
	}
	if item.MinPodsAvailableCount == nil {
		return false
	}
	if item.MinPodsAvailableRatio == nil {
		return false
	}
	if item.DiscreteValueOutOfList == nil {
		return false
	}
	if !isDiscreteValueOutOfListDefaulted(item.DiscreteValueOutOfList) {
		return false
	}

	return true
}

// isDiscreteValueOutOfListDefaulted used to check if a DiscreteValueOutOfList is already defaulted
func isDiscreteValueOutOfListDefaulted(item *DiscreteValueOutOfList) bool {
	if item.BadValues == nil {
		return false
	}
	if item.GoodValues == nil {
		return false
	}
	if item.MinimumActivityCount == nil {
		return false
	}
	if item.TolerancePercent == nil {
		return false
	}
	return true
}
