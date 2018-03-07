package v1

import "time"

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

func NewUInt(val uint) *uint {
	output := new(uint)
	*output = val
	return output
}
