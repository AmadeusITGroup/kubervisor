package generator

import (
	"math/rand"

	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/api"
)

func init() {
	rand.Seed(42)
}

const (
	// ParisCityCode Paris City code
	ParisCityCode api.CityCode = "PAR"
	// LondonCityCode Paris City code
	LondonCityCode api.CityCode = "LON"
	// CopenhagenCityCode Paris City code
	CopenhagenCityCode api.CityCode = "CPH"
)
