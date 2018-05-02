package controller

import (
	"net/http"

	kubervisorclient "github.com/amadeusitgroup/kubervisor/pkg/client"
	bclient "github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
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

	nbWorker uint32

	logger *zap.Logger
}

var _ Initializer = &Config{}

// NewConfig returns new Config struct instance
func NewConfig(logger *zap.Logger) *Config {
	return &Config{logger: logger}
}

// SetLogger set or change the logger in the config
func (c *Config) SetLogger(logger *zap.Logger) {
	c.logger = logger
}

// AddFlags add cobra flags to populate Config
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KubeConfigFile, "kubeconfig", c.KubeConfigFile, "Location of kubecfg file for access to kubernetes master service")
	fs.StringVar(&c.Master, "master", c.Master, "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	fs.StringVar(&c.ListenAddr, "addr", "0.0.0.0:8086", "listen address of the http server which serves kubernetes probes and prometheus endpoints")
	fs.Uint32Var(&c.nbWorker, "nbworker", nbWorkerDefault, "Number of running workers in the controller")
	fs.BoolVar(&c.Debug, "debug", c.Debug, "used to activate debug logs")
}

//RegisterAPI registers the apiextension in kubernetes apiserver
func (c *Config) RegisterAPI() error {
	sugar := c.logger.Sugar()
	kubeConfig, err := initKubeConfig(c)
	if err != nil {
		sugar.Fatalf("Unable to init kubervisor controller: %v", err)
		return err
	}

	extClient, err := apiextensionsclient.NewForConfig(kubeConfig)
	if err != nil {
		sugar.Fatalf("Unable to init clientset from kubeconfig:%v", err)
		return err
	}

	_, err = kubervisorclient.DefineKubervisorResources(extClient)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		sugar.Fatalf("Unable to define KubervisorService resource:%v", err)
		return err
	}
	return nil
}

//InitClients returns the kube client and the extension
func (c *Config) InitClients() (kubeClient clientset.Interface, breakerClient bclient.Interface, leaderElectionClient clientset.Interface, err error) {
	sugar := c.logger.Sugar()
	kubeConfig, err := initKubeConfig(c)
	if err != nil {
		sugar.Fatalf("Unable to init kubervisor controller: %v", err)
	}

	kubeClient, err = clientset.NewForConfig(kubeConfig)
	if err != nil {
		sugar.Fatalf("Unable to initialize kubeClient:%v", err)
	}

	leaderElectionClient, err = clientset.NewForConfig(restclientset.AddUserAgent(kubeConfig, "leader-election"))
	if err != nil {
		sugar.Fatalf("Unable to initialize leaderElectionClient:%v", err)
	}

	breakerClient, err = kubervisorclient.NewClient(kubeConfig)
	if err != nil {
		sugar.Fatalf("Unable to init kubervisor.clientset from kubeconfig:%v", err)
	}
	return kubeClient, breakerClient, leaderElectionClient, err
}

//Logger returns the logger associated to the configuration
func (c *Config) Logger() *zap.Logger {
	return c.logger
}

//NbWorker returns the configured number of workers
func (c *Config) NbWorker() uint32 {
	return c.nbWorker
}

//HTTPServer returns the http server associated to the configuration
func (c *Config) HTTPServer() *http.Server {
	return &http.Server{Addr: c.ListenAddr}
}
