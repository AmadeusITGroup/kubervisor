package utils

import (
	"github.com/prometheus/client_golang/prometheus"
)

// NewPriceHistogram return a new HistogramVec for pricing
func NewPriceHistogram() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pricer_price",
		Help:    "solution price distributions.",
		Buckets: prometheus.LinearBuckets(0, 100, 100),
	}, []string{"provider", "od"})
}
