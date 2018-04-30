package message

import (
	"time"

	"github.com/amadeusitgroup/kubervisor/examples/demo/pkg/api"
)

// Request request for the price service
type Request struct {
	OD              api.OriginDestination `json:"od,omitempty"`
	Date            time.Time             `json:"date,omitempty"`
	PassengerNumber uint                  `json:"passenger_number,omitempty"`
}
