package framework

import (
	"fmt"

	"github.com/amadeusitgroup/kubervisor/pkg/client"
	"github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Framework stores necessary info to run e2e
type Framework struct {
	KubeConfig *rest.Config
}

type frameworkContextType struct {
	KubeConfigPath        string
	ContainerRegistryHost string
}

// FrameworkContext stores globally the framework context
var FrameworkContext frameworkContextType

// NewFramework creates and initializes the a Framework struct
func NewFramework() (*Framework, error) {
	Logf("KubeconfigPath-> %q", FrameworkContext.KubeConfigPath)
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", FrameworkContext.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve kubeConfig:%v", err)
	}
	return &Framework{
		KubeConfig: kubeConfig,
	}, nil
}

func (f *Framework) kubeClient() (clientset.Interface, error) {
	return clientset.NewForConfig(f.KubeConfig)
}

func (f *Framework) kubervisorClient() (versioned.Interface, error) {
	c, err := client.NewClient(f.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubervisor client:%v", err)
	}
	return c, err
}
