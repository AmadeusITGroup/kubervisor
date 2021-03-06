package v1alpha1

import (
	"testing"
)

func Test_isDiscreteValueOutOfListDefaulted(t *testing.T) {
	type args struct {
		item *DiscreteValueOutOfList
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Already defaulted",
			args: args{
				item: DefaultDiscreteValueOutOfList(&DiscreteValueOutOfList{}),
			},
			want: true,
		},
		{
			name: "missing tolerance percent",
			args: args{
				item: &DiscreteValueOutOfList{
					PrometheusService: "",
					PromQL:            "",
					Key:               "",
					PodNameKey:        "",
					GoodValues:        []string{},
					BadValues:         []string{},
					//TolerancePercent:     NewUInt(1),
					MinimumActivityCount: NewUInt(1),
				},
			},
			want: false,
		},
		{
			name: "missing MinimumActivityCount",
			args: args{
				item: &DiscreteValueOutOfList{
					PrometheusService: "",
					PromQL:            "",
					Key:               "",
					PodNameKey:        "",
					GoodValues:        []string{},
					BadValues:         []string{},
					TolerancePercent:  NewUInt(1),
					//MinimumActivityCount: NewUInt(1),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDiscreteValueOutOfListDefaulted(tt.args.item); got != tt.want {
				t.Errorf("isDiscreteValueOutOfListDefaulted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isBreakerStrategyDefaulted(t *testing.T) {

	activatorNotDefaulted := DefaultBreakerStrategy(&BreakerStrategy{})
	activatorNotDefaulted.Activator = &ActivatorStrategy{}

	type args struct {
		item *BreakerStrategy
	}
	tests := []struct {
		name string
		args args
		want bool
	}{

		{
			name: "already defaulted",
			args: args{
				item: DefaultBreakerStrategy(&BreakerStrategy{}),
			},
			want: true,
		},
		{
			name: "activator in strategy not defaulted",
			args: args{
				item: activatorNotDefaulted,
			},
			want: false,
		},
		{
			name: "missing EvaluationPeriod",
			args: args{
				item: &BreakerStrategy{
					MinPodsAvailableCount:  NewUInt(1),
					MinPodsAvailableRatio:  NewUInt(70),
					DiscreteValueOutOfList: DefaultDiscreteValueOutOfList(&DiscreteValueOutOfList{}),
				},
			},
			want: false,
		},
		{
			name: "missing MinPodsAvailableCount",
			args: args{
				item: &BreakerStrategy{
					EvaluationPeriod:       NewFloat64(1.0),
					MinPodsAvailableRatio:  NewUInt(70),
					DiscreteValueOutOfList: DefaultDiscreteValueOutOfList(&DiscreteValueOutOfList{}),
				},
			},
			want: true,
		},
		{
			name: "missing MinPodsAvailableCount and MinPodsAvailableRatio",
			args: args{
				item: &BreakerStrategy{
					EvaluationPeriod:       NewFloat64(1.0),
					DiscreteValueOutOfList: DefaultDiscreteValueOutOfList(&DiscreteValueOutOfList{}),
				},
			},
			want: false,
		},
		{
			name: "missing MinPodsAvailableRatio",
			args: args{
				item: &BreakerStrategy{
					EvaluationPeriod:       NewFloat64(1.0),
					MinPodsAvailableCount:  NewUInt(1),
					DiscreteValueOutOfList: DefaultDiscreteValueOutOfList(&DiscreteValueOutOfList{}),
				},
			},
			want: true,
		},
		{
			name: "missing DiscreteValueOutOfList values",
			args: args{
				item: &BreakerStrategy{
					EvaluationPeriod:       NewFloat64(1.0),
					MinPodsAvailableCount:  NewUInt(1),
					MinPodsAvailableRatio:  NewUInt(70),
					DiscreteValueOutOfList: &DiscreteValueOutOfList{},
				},
			},
			want: false,
		},
		{
			name: "missing ContinuousValueDeviation values",
			args: args{
				item: &BreakerStrategy{
					EvaluationPeriod:         NewFloat64(1.0),
					MinPodsAvailableCount:    NewUInt(1),
					MinPodsAvailableRatio:    NewUInt(70),
					ContinuousValueDeviation: &ContinuousValueDeviation{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBreakerStrategyDefaulted(tt.args.item); got != tt.want {
				t.Errorf("isBreakerStrategyDefaulted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isActivatorStrategyDefaulted(t *testing.T) {
	type args struct {
		item *ActivatorStrategy
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "already defaulted",
			args: args{
				item: DefaultActivatorStrategy(&ActivatorStrategy{}),
			},
			want: true,
		},
		{
			name: "missing Mode",
			args: args{
				item: &ActivatorStrategy{
					//Mode: ActivatorStrategyModePeriodic,
					MaxRetryCount: NewUInt(2),
					MaxPauseCount: NewUInt(20),
					Period:        NewFloat64(1.0),
				},
			},
			want: false,
		},
		{
			name: "missing MaxRetryCount",
			args: args{
				item: &ActivatorStrategy{
					Mode: ActivatorStrategyModePeriodic,
					//MaxRetryCount: NewUInt(2),
					MaxPauseCount: NewUInt(20),
					Period:        NewFloat64(1.0),
				},
			},
			want: false,
		},
		{
			name: "missing MaxPauseCount",
			args: args{
				item: &ActivatorStrategy{
					Mode:          ActivatorStrategyModePeriodic,
					MaxRetryCount: NewUInt(2),
					//MaxPauseCount: NewUInt(20),
					Period: NewFloat64(1.0),
				},
			},
			want: false,
		},
		{
			name: "missing Period",
			args: args{
				item: &ActivatorStrategy{
					Mode:          ActivatorStrategyModePeriodic,
					MaxRetryCount: NewUInt(2),
					MaxPauseCount: NewUInt(20),
					//Period:        NewFloat64(1.0),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isActivatorStrategyDefaulted(tt.args.item); got != tt.want {
				t.Errorf("isActivatorStrategyDefaulted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsKubervisorServiceDefaulted(t *testing.T) {
	type args struct {
		bc *KubervisorService
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "already defaulted",
			args: args{
				bc: &KubervisorService{
					Spec: KubervisorServiceSpec{
						DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
						Breakers:         []BreakerStrategy{*DefaultBreakerStrategy(&BreakerStrategy{})},
					},
				},
			},
			want: true,
		},
		{
			name: "Activator not defaulted",
			args: args{
				bc: &KubervisorService{
					Spec: KubervisorServiceSpec{
						Breakers: []BreakerStrategy{*DefaultBreakerStrategy(&BreakerStrategy{})},
					},
				},
			},
			want: false,
		},
		{
			name: "Breaker not defaulted",
			args: args{
				bc: &KubervisorService{
					Spec: KubervisorServiceSpec{
						Breakers:         []BreakerStrategy{BreakerStrategy{}},
						DefaultActivator: *DefaultActivatorStrategy(&ActivatorStrategy{}),
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKubervisorServiceDefaulted(tt.args.bc); got != tt.want {
				t.Errorf("IsKubervisorServiceDefaulted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKubervisorService(t *testing.T) {
	bc := &KubervisorService{}
	bc = DefaultKubervisorService(bc)

	if !IsKubervisorServiceDefaulted(bc) {
		t.Errorf("KubervisorService is not defaulted properly")
	}
}
