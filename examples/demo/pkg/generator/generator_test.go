package generator

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/api"
)

func TestCalculateODPrice(t *testing.T) {
	type args struct {
		kmPrice     float32
		randPercent uint32
		od          api.OriginDestination
	}
	tests := []struct {
		name    string
		args    args
		minWant float64
		maxWant float64
	}{
		{
			name: "no random price",
			args: args{
				kmPrice:     1.0,
				randPercent: 0,
				od: api.OriginDestination{
					Distance: 100,
				},
			},
			minWant: 100,
			maxWant: 100,
		},
		{
			name: "10% random",
			args: args{
				kmPrice:     1.0,
				randPercent: 0,
				od: api.OriginDestination{
					Distance: 100,
				},
			},
			minWant: 90,
			maxWant: 110,
		},
		{
			name: "kmprice 0",
			args: args{
				kmPrice:     0.0,
				randPercent: 0,
				od: api.OriginDestination{
					Distance: 100,
				},
			},
			minWant: 0,
			maxWant: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateODPrice(tt.args.kmPrice, tt.args.randPercent, tt.args.od); !(got >= tt.minWant && got <= tt.maxWant) {
				t.Errorf("CalculateODPrice() = %v, minWant %v, maxWant %v", got, tt.minWant, tt.maxWant)
			}
		})
	}
}

func TestGenerateRoutes(t *testing.T) {
	now := time.Now()
	schedules1 := []time.Time{now}
	parlonOD := api.OriginDestination{Origin: ParisCityCode, Destination: LondonCityCode, Distance: 100}
	ods1 := []api.OriginDestination{parlonOD}

	type args struct {
		schedules    []time.Time
		providerName string
		kmPrice      float32
		randPercent  uint32
		ods          []api.OriginDestination
	}
	tests := []struct {
		name string
		args args
		want []api.Route
	}{
		{
			name: "basic test",
			args: args{
				schedules:    schedules1,
				providerName: "1A",
				kmPrice:      1.0,
				randPercent:  0,
				ods:          ods1,
			},
			want: []api.Route{
				{
					ID: "PARLON",
					Segments: []api.Flight{
						{Date: now, OD: parlonOD, ID: fmt.Sprintf("%s%s%s%d", "1A", parlonOD.Origin, parlonOD.Destination, now.Hour()), Provider: "1A"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateRoutes(tt.args.schedules, tt.args.providerName, tt.args.kmPrice, tt.args.randPercent, tt.args.ods); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateRoutes() = %v, want %v", got, tt.want)
			}
		})
	}
}
