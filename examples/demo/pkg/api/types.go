package api

import (
	"fmt"
	"time"
)

type CityCode string

type OriginDestination struct {
	Origin      CityCode `json:"origin"`
	Destination CityCode `json:"destination"`
	Distance    int32    `json:"distance,omitempty"`
}

type Route struct {
	ID       string   `json:"id,omitempty"`
	Segments []Flight `json:"segments,omitempty"`
}

type Flight struct {
	ID       string            `json:"id,omitempty"`
	OD       OriginDestination `json:"od,omitempty"`
	Date     time.Time         `json:"date,omitempty"`
	Provider string            `json:"provider,omitempty"`
	Price    Price             `json:"price,omitempty"`
}

type Solution struct {
	Route       Route   `json:"route,omitempty"`
	TotalPrice  Price   `json:"total_price,omitempty"`
	DetailPrice []Price `json:"detail_price,omitempty"`
}

type Price struct {
	Price     float64 `json:"price"`
	Currency  string  `json:"currency,omitempty"`
	SegmentID *string `json:"segment_id,omitempty"`
}

type Error struct {
	Code        int    `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

type Warning struct {
	Code        int    `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

func (od OriginDestination) String() string {
	return fmt.Sprintf("%s%s", od.Origin, od.Destination)
}
