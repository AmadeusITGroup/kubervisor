package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/api"
)

// GetODs returns a static list of all OriginDestination in the system
func GetODs() []api.OriginDestination {
	ods := []api.OriginDestination{
		// Paris - London
		{
			Origin:      ParisCityCode,
			Destination: LondonCityCode,
			Distance:    343,
		},
		{
			Origin:      LondonCityCode,
			Destination: ParisCityCode,
			Distance:    343,
		},

		// Paris - Copenhagen
		{
			Origin:      ParisCityCode,
			Destination: CopenhagenCityCode,
			Distance:    1027,
		},
		{
			Origin:      CopenhagenCityCode,
			Destination: ParisCityCode,
			Distance:    1027,
		},

		// London - Copenhagen
		{
			Origin:      LondonCityCode,
			Destination: CopenhagenCityCode,
			Distance:    957,
		},
		{
			Origin:      CopenhagenCityCode,
			Destination: LondonCityCode,
			Distance:    957,
		},
	}

	return ods
}

// GenerateRandHourMin return a time.Time with a random Hour-Min on top of the schedule time.Time argument
func GenerateRandHourMin(schedule time.Time) time.Time {
	randHour := rand.Int31n(23)
	randMin := rand.Int31n(59)

	return time.Date(schedule.Year(), schedule.Month(), schedule.Day(), int(randHour), int(randMin), 0, 0, schedule.Location())
}

// GenerateSchedules generate a number of schedules for a specific day
func GenerateSchedules(nbSchedule int, day time.Time) []time.Time {
	times := []time.Time{}
	for i := 0; i < nbSchedule; i++ {
		s := GenerateRandHourMin(day)
		times = append(times, s)
	}

	return times
}

// GenerateRoutes generate Route slice from a set of config parameters and a list of ODs
func GenerateRoutes(schedules []time.Time, providerName string, kmPrice float32, randPercent uint32, ods []api.OriginDestination) []api.Route {
	routes := []api.Route{}

	for _, od := range ods {
		for _, schedule := range schedules {
			newRoute := api.Route{
				ID: fmt.Sprintf("%s%s", od.Origin, od.Destination),
				Segments: []api.Flight{
					{
						Date:     schedule,
						OD:       od,
						Provider: providerName,
						ID:       fmt.Sprintf("%s%s%s%d", providerName, od.Origin, od.Destination, schedule.Hour()),
					},
				},
			}
			routes = append(routes, newRoute)
		}
	}

	return routes
}

// CalculateODPrice calculate the Price of a Flight depending of the price by km, the random price persentage for an OD
func CalculateODPrice(kmPrice float32, randPercent uint32, od api.OriginDestination) float64 {
	price := float64(kmPrice) * float64(od.Distance)

	percentPrice := price * float64(randPercent) / 100
	randNumber := rand.Int31n(200)

	return price - percentPrice + (percentPrice*float64(randNumber))/100
}
