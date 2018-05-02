package comparator

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/spf13/pflag"
)

// Config comparator process configuration
type Config struct {
	Port         string
	Providers    []string
	PromRegistry *prometheus.Registry
}

// NewConfig returns new config instance
func NewConfig(reg *prometheus.Registry) *Config {
	return &Config{
		Port:         "8080",
		PromRegistry: reg,
		Providers:    []string{},
	}
}

// AddFlags add cobra flags to populate Config
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Port, "port", c.Port, "http server port")
	fs.StringArrayVar(&c.Providers, "provider", c.Providers, "provider service")
}
