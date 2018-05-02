package main

import (
	goflag "flag"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/amadeusitgroup/kubervisor/pkg/controller"
)

var kubeClient clientset.Interface
var sugar *zap.SugaredLogger
var config *controller.Config
var serverAddr string
var namespace string
var selector string

func initMockFlag(fs *pflag.FlagSet) {
	fs.StringVar(&serverAddr, "serverAddr", "0.0.0.0:8080", "listen address of the http server which serves kubernetes probes and prometheus endpoints")
	fs.StringVar(&namespace, "namespace", "default", "namespace in which we need to list the pod")
	fs.StringVar(&selector, "selector", "", "selector used to list the pod")
}

func initGlobals() {
	logger, _ := zap.NewProduction()
	config = controller.NewConfig(logger)
	config.AddFlags(pflag.CommandLine)
	initMockFlag(pflag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	goflag.CommandLine.Parse([]string{})

	if config.Debug {
		l, _ := zap.NewDevelopment()
		config.SetLogger(l)
	}

	sugar = config.Logger().Sugar()

	initKubeClient()
}

func initKubeClient() {
	kubeConfig, err := initKubeConfig()
	if err != nil || kubeConfig == nil {
		sugar.Fatalf("Unable to init kubeconfig: %v", err)
	}

	kubeClient, err = clientset.NewForConfig(kubeConfig)
	if err != nil || kubeClient == nil {
		sugar.Fatalf("Unable to initialize kubeClient:%v", err)
	}
}

func initKubeConfig() (*rest.Config, error) {
	if len(config.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}
