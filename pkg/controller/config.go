package controller

import (
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	nbWorkerDefault = 3
)

// Config represent the kubevisor binary configuration
type Config struct {
	KubeConfigFile string
	Master         string
	ListenAddr     string
	Debug          bool

	NbWorker uint32

	Logger *zap.Logger
}

// NewConfig returns new Config struct instance
func NewConfig(logger *zap.Logger) *Config {
	return &Config{Logger: logger}
}

// AddFlags add cobra flags to populate Config
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KubeConfigFile, "kubeconfig", c.KubeConfigFile, "Location of kubecfg file for access to kubernetes master service")
	fs.StringVar(&c.Master, "master", c.Master, "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	fs.StringVar(&c.ListenAddr, "addr", "0.0.0.0:8086", "listen address of the http server which serves kubernetes probes and prometheus endpoints")
	fs.Uint32Var(&c.NbWorker, "nbworker", nbWorkerDefault, "Number of running workers in the controller")
	fs.BoolVar(&c.Debug, "debug", c.Debug, "used to activate debug logs")
}
