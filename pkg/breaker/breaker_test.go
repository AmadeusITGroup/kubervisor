package breaker

import (
	"testing"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
)

func TestBreakerImpl_computeMinAvailablePods(t *testing.T) {
	type fields struct {
		breakerStrategyConfig v1.BreakerStrategy
	}
	type args struct {
		podUnderSelectorCount int
	}
	tests := []struct {
		name                  string
		fields                fields
		podUnderSelectorCount int
		want                  int
	}{
		{
			name: "zero",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{},
			},
			podUnderSelectorCount: 10,
			want: 0,
		},
		{
			name: "count3",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableCount: v1.NewUInt(3),
				},
			},
			podUnderSelectorCount: 10,
			want: 3,
		},
		{
			name: "ratio5",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableRatio: v1.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 5,
		},
		{
			name: "ratio5count3",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableCount: v1.NewUInt(3),
					MinPodsAvailableRatio: v1.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 5,
		},
		{
			name: "ratio5count8",
			fields: fields{
				breakerStrategyConfig: v1.BreakerStrategy{
					MinPodsAvailableCount: v1.NewUInt(8),
					MinPodsAvailableRatio: v1.NewUInt(50),
				},
			},
			podUnderSelectorCount: 10,
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BreakerImpl{
				breakerStrategyConfig: tt.fields.breakerStrategyConfig,
			}
			if got := b.computeMinAvailablePods(tt.podUnderSelectorCount); got != tt.want {
				t.Errorf("BreakerImpl.computeMinAvailablePods() = %v, want %v", got, tt.want)
			}
		})
	}
}
