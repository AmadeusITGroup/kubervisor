package v1

//DefaultDiscreteValueOutOfList injecting default value for the struct
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

func NewUInt(val uint) *uint {
	output := new(uint)
	*output = val
	return output
}
