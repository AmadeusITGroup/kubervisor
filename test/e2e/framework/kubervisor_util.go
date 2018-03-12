package framework

import (
	"fmt"
	// imported for test
	. "github.com/onsi/gomega"

	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/client/clientset/versioned"
)

// BuildAndSetClients builds and initilize rediscluster and kube client
func BuildAndSetClients() (versioned.Interface, clientset.Interface) {
	f, err := NewFramework()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(f).ShouldNot(BeNil())

	kubeClient, err := f.kubeClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubeClient).ShouldNot(BeNil())
	Logf("Check whether RedisCluster resource is registered...")

	kubervisorClient, err := f.kubervisorClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubervisorClient).ShouldNot(BeNil())
	return kubervisorClient, kubeClient
}

// NewBreakerConfig return new instance of a BreakerConfig
func NewBreakerConfig(name string) *v1.BreakerConfig {
	return &v1.BreakerConfig{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.BreakerConfigSpec{
			Activator: *v1.DefaultActivatorStrategy(&v1.ActivatorStrategy{}),
			Breaker:   *v1.DefaultBreakerStrategy(&v1.BreakerStrategy{}),
		},
	}
}

// CreateBreakerConfig is an higher order func that returns the func to create a BreakerConfig
func CreateBreakerConfig(client versioned.Interface, bc *v1.BreakerConfig, namespace string) func() error {
	return func() error {
		if _, err := client.BreakerV1().BreakerConfigs(namespace).Create(bc); err != nil {
			Warningf("cannot create BreakerConfig %s/%s: %v", namespace, bc.Name, err)
			return err
		}
		Logf("BreakerConfig created")
		return nil
	}
}

// IsBreakerConfigCreated is an higher order func that returns the func to create a BreakerConfig
func IsBreakerConfigCreated(client versioned.Interface, name, namespace string) func() error {
	return func() error {
		bc, err := client.BreakerV1().BreakerConfigs(namespace).Get(name, kmetav1.GetOptions{})
		if err != nil {
			Warningf("cannot get BreakerConfig %s/%s: %v", namespace, name, err)
			return err
		}
		if bc == nil {
			return fmt.Errorf("BreakerConfig  %s/%s is nil", namespace, name)
		}
		return nil
	}
}
