package pricer

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/spf13/pflag"
)

// Config pricer process configuration
type Config struct {
	Port             string
	Provider         string
	PromRegistry     *prometheus.Registry
	KmPrice          float32
	RandPricePercent uint32
}

// NewConfig returns new config instance
func NewConfig(reg *prometheus.Registry) *Config {
	return &Config{
		Port:             "8080",
		PromRegistry:     reg,
		Provider:         "1A",
		RandPricePercent: 30,
		KmPrice:          1.0,
	}
}

// AddFlags add cobra flags to populate Config
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Port, "port", c.Port, "http server port")
	fs.StringVar(&c.Provider, "provider", c.Provider, "provider name")
	fs.Uint32Var(&c.RandPricePercent, "rand-price", c.RandPricePercent, "the price randomnisation persentage")
	fs.Float32Var(&c.KmPrice, "km-price", c.KmPrice, "price by km")
}
