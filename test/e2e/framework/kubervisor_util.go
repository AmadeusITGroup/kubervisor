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

// NewKubervisorService return new instance of a KubervisorService
func NewKubervisorService(name string) *v1.KubervisorService {
	return &v1.KubervisorService{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.KubervisorServiceSpec{
			Activator: *v1.DefaultActivatorStrategy(&v1.ActivatorStrategy{}),
			Breaker:   *v1.DefaultBreakerStrategy(&v1.BreakerStrategy{}),
		},
	}
}

// CreateKubervisorService is an higher order func that returns the func to create a KubervisorService
func CreateKubervisorService(client versioned.Interface, bc *v1.KubervisorService, namespace string) func() error {
	return func() error {
		if _, err := client.BreakerV1().KubervisorServices(namespace).Create(bc); err != nil {
			Warningf("cannot create KubervisorService %s/%s: %v", namespace, bc.Name, err)
			return err
		}
		Logf("KubervisorService created")
		return nil
	}
}

// IsKubervisorServiceCreated is an higher order func that returns the func to create a KubervisorService
func IsKubervisorServiceCreated(client versioned.Interface, name, namespace string) func() error {
	return func() error {
		bc, err := client.BreakerV1().KubervisorServices(namespace).Get(name, kmetav1.GetOptions{})
		if err != nil {
			Warningf("cannot get KubervisorService %s/%s: %v", namespace, name, err)
			return err
		}
		if bc == nil {
			return fmt.Errorf("KubervisorService  %s/%s is nil", namespace, name)
		}
		return nil
	}
}
