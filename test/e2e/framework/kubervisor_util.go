package framework

import (
	"fmt"
	// imported for test
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	kv1 "k8s.io/api/core/v1"
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

func CheckEndpointsCount(client clientset.Interface, name, namespace string, countReadyTarget int32, countNotReadyTarget int32) func() error {
	return func() error {
		ep, err := client.CoreV1().Endpoints(namespace).Get("busybox", kmetav1.GetOptions{})
		if err != nil {
			return err
		}
		if ep == nil {
			return fmt.Errorf("No Eendpoint")
		}

		var countReady, countNotReady int32
		for _, subset := range ep.Subsets {
			countReady += int32(len(subset.Addresses) * len(subset.Ports))
			countNotReady += int32(len(subset.NotReadyAddresses) * len(subset.Ports))
		}

		if countReadyTarget == countReady && countNotReadyTarget == countNotReady {
			return nil
		}

		return fmt.Errorf("Bad endpoint count Ready %d/%d and notReady %d/%d", countReady, countReadyTarget, countNotReady, countNotReadyTarget)
	}

}

// NewUInt return a pointer to a uint
func NewInt32(val int32) *int32 {
	output := new(int32)
	*output = val
	return output
}
func CreateBusyBox(client clientset.Interface, namespace string) func() error {
	return func() error {
		deployment := appsv1.Deployment{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:   "busybox",
				Labels: map[string]string{"purpose": "e2eTest"},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: NewInt32(5),
				Selector: &kmetav1.LabelSelector{
					MatchLabels: map[string]string{"app": "busybox"},
				},
				Template: kv1.PodTemplateSpec{
					ObjectMeta: kmetav1.ObjectMeta{
						Labels: map[string]string{"app": "busybox"},
					},
					Spec: kv1.PodSpec{
						Containers: []kv1.Container{
							{
								Name:            "busybox",
								Image:           "busybox",
								Command:         []string{"sleep", "3600"},
								ImagePullPolicy: "IfNotPresent",
							},
						},
					},
				},
			},
		}
		if _, err := client.Apps().Deployments(namespace).Create(&deployment); err != nil {
			return err
		}

		service := kv1.Service{
			ObjectMeta: kmetav1.ObjectMeta{
				Name: "busybox",
			},
			Spec: kv1.ServiceSpec{
				Selector: map[string]string{"app": "busybox"},
				Ports: []kv1.ServicePort{
					{Port: 80},
				},
			},
		}
		if _, err := client.CoreV1().Services(namespace).Create(&service); err != nil {
			return err
		}

		return nil
	}
}

func CreateNamespace(client clientset.Interface, namespace string) func() error {
	return func() error {
		ns := kv1.Namespace{
			ObjectMeta: kmetav1.ObjectMeta{
				Name: namespace,
			},
		}
		if _, err := client.Core().Namespaces().Create(&ns); err != nil {
			return err
		}
		return nil
	}
}

func DeleteNamespace(client clientset.Interface, namespace string) func() error {
	return func() error {
		if err := client.Core().Namespaces().Delete(namespace, nil); err != nil {
			return err
		}
		return nil
	}
}
