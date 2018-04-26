package framework

import (
	"fmt"

	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"

	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/labels"
	// imported for test
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	kv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
		if _, err := client.KubervisorV1().KubervisorServices(namespace).Create(bc); err != nil {
			Warningf("cannot create KubervisorService %s/%s: %v", namespace, bc.Name, err)
			return err
		}
		Logf("KubervisorService created")
		return nil
	}
}

// DeleteKubervisorService is an higher order func that returns the func to create a KubervisorService
func DeleteKubervisorService(client versioned.Interface, name, namespace string) func() error {
	return func() error {
		if err := client.KubervisorV1().KubervisorServices(namespace).Delete(name, nil); err != nil {
			Warningf("cannot delete KubervisorService %s/%s: %v", namespace, name, err)
			return err
		}
		Logf("KubervisorService created")
		return nil
	}
}

//CheckKubervisorServiceStatus validate the status part of the kubervisor crd
func CheckKubervisorServiceStatus(client versioned.Interface, name, namespace string, managed, breaked, paused, unknown uint32) func() error {
	return func() error {
		bc, err := client.KubervisorV1().KubervisorServices(namespace).Get(name, kmetav1.GetOptions{})
		if err != nil {
			Warningf("cannot delete KubervisorService %s/%s: %v", namespace, name, err)
			return err
		}
		if bc.Status.Breaker.NbPodsManaged != managed {
			return fmt.Errorf("Bad managed count: expect %d got %d", managed, bc.Status.Breaker.NbPodsManaged)
		}
		if bc.Status.Breaker.NbPodsBreaked != breaked {
			return fmt.Errorf("Bad breaked count: expect %d got %d", breaked, bc.Status.Breaker.NbPodsBreaked)
		}
		if bc.Status.Breaker.NbPodsPaused != paused {
			return fmt.Errorf("Bad paused count: expect %d got %d", paused, bc.Status.Breaker.NbPodsPaused)
		}
		if bc.Status.Breaker.NbPodsUnknown != unknown {
			return fmt.Errorf("Bad unknown count: expect %d got %d", unknown, bc.Status.Breaker.NbPodsUnknown)
		}

		return nil
	}

}

// IsKubervisorServiceCreated is an higher order func that returns the func to create a KubervisorService
func IsKubervisorServiceCreated(client versioned.Interface, name, namespace string) func() error {
	return func() error {
		bc, err := client.KubervisorV1().KubervisorServices(namespace).Get(name, kmetav1.GetOptions{})
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

//CheckEndpointsCount count the number of ready / notready endpoints
func CheckEndpointsCount(client clientset.Interface, name, namespace string, countReadyTarget int32, countNotReadyTarget int32) func() error {
	return func() error {
		ep, err := client.CoreV1().Endpoints(namespace).Get(name, kmetav1.GetOptions{})
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
				Selector: map[string]string{"app": "busybox", labeling.LabelTrafficKey: string(labeling.LabelTrafficYes)},
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

func E2ERBAC(client clientset.Interface, namespace string) func() error {
	return func() error {

		roleName := "e2eRole"

		role := rbacv1.Role{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      roleName,
				Namespace: namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "services"},
					Verbs:     []string{"*"},
				},
			},
		}
		if _, err := client.Rbac().Roles(namespace).Create(&role); err != nil {
			return err
		}

		rolebiding := rbacv1.RoleBinding{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      "e2eRoleBinding",
				Namespace: namespace,
			},

			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     roleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "default",
					Namespace: namespace,
				},
			},
		}

		if _, err := client.Rbac().RoleBindings(namespace).Create(&rolebiding); err != nil {
			return err
		}
		return nil
	}
}

func CreateCustomAnamalyDetector(client clientset.Interface, namespace, targetNamespace string, selector labels.Selector) func() error {
	return func() error {

		deployment := appsv1.Deployment{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:   "customanomalydetector",
				Labels: map[string]string{"purpose": "e2eTest"},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: NewInt32(1),
				Selector: &kmetav1.LabelSelector{
					MatchLabels: map[string]string{"kubervisor": "customanomalydetector"},
				},
				Template: kv1.PodTemplateSpec{
					ObjectMeta: kmetav1.ObjectMeta{
						Labels: map[string]string{"kubervisor": "customanomalydetector"},
					},
					Spec: kv1.PodSpec{
						Containers: []kv1.Container{
							{
								Name:            "customanomalydetector",
								Image:           "podkubervisor/customanomalydetector",
								Args:            []string{"--namespace=" + targetNamespace, "--selector=" + selector.String()},
								ImagePullPolicy: "IfNotPresent",
							},
						},
					},
				},
			},
		}
		if _, err := client.Apps().Deployments(targetNamespace).Create(&deployment); err != nil {
			return err
		}

		service := kv1.Service{
			ObjectMeta: kmetav1.ObjectMeta{
				Name: "customanomalydetector",
			},
			Spec: kv1.ServiceSpec{
				Selector: map[string]string{"kubervisor": "customanomalydetector"},
				Ports: []kv1.ServicePort{
					{Port: 80, TargetPort: intstr.FromInt(8080)},
				},
			},
		}
		if _, err := client.CoreV1().Services(targetNamespace).Create(&service); err != nil {
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

func ExpectCountPod(client clientset.Interface, namespace string, selector labels.Selector, count int) func() error {
	return func() error {
		options := kmetav1.ListOptions{
			LabelSelector: selector.String(),
		}
		pods, err := client.Core().Pods(namespace).List(options)
		if err != nil {
			return err
		}
		if len(pods.Items) != count {
			return fmt.Errorf("Bad pod count: expect %d got %d", count, len(pods.Items))
		}
		return nil
	}
}

func TagPod(client clientset.Interface, namespace string, selector labels.Selector, count int, tags labels.Set) func() error {
	return func() error {
		options := kmetav1.ListOptions{
			LabelSelector: selector.String(),
		}
		pods, err := client.Core().Pods(namespace).List(options)
		if err != nil {
			return err
		}

		match := []kv1.Pod{}
		unmatch := []kv1.Pod{}

		for _, p := range pods.Items {
			if tags.AsSelector().Matches(labels.Set(p.GetLabels())) {
				match = append(match, p)
			} else {
				unmatch = append(unmatch, p)
			}
		}
		switch {
		case len(match) > count:
			//remove tags
			for len(match) > count {
				p := match[0]
				match = match[1:]
				for k := range tags {
					delete(p.Labels, k)
				}
				if _, err := client.Core().Pods(namespace).Update(&p); err != nil {
					return err
				}
				unmatch = append(unmatch, p)

			}
		case len(match) < count:
			//add tags
			for i := 0; i < len(unmatch) && len(match) < count; i++ {
				p := unmatch[i]
				for k, v := range tags {
					p.Labels[k] = v
				}
				if _, err := client.Core().Pods(namespace).Update(&p); err != nil {
					return err
				}
				match = append(match, p)
			}
		}
		return nil
	}
}
