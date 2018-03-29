package e2e

import (
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"

	// for test lisibility
	gink "github.com/onsi/ginkgo"
	// for test lisibility
	gom "github.com/onsi/gomega"

	"github.com/amadeusitgroup/podkubervisor/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/podkubervisor/test/e2e/framework"
)

var kubervisorClient versioned.Interface
var kubeClient clientset.Interface

const testNs = "e2e"

var _ = gink.BeforeSuite(func() {
	kubervisorClient, kubeClient = framework.BuildAndSetClients()
	gom.Eventually(framework.CreateNamespace(kubeClient, testNs), "5s", "1s").ShouldNot(gom.HaveOccurred())
})

var _ = gink.AfterSuite(func() {
	gom.Eventually(framework.DeleteNamespace(kubeClient, testNs), "5s", "1s").ShouldNot(gom.HaveOccurred())
})

var _ = gink.Describe("KubervisorService CRUD", func() {
	gink.It("should create a KubervisorService", func() {
		bc := framework.NewKubervisorService("foo")
		gom.Eventually(framework.CreateKubervisorService(kubervisorClient, bc, testNs), "5s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.IsKubervisorServiceCreated(kubervisorClient, bc.Name, testNs), "5s", "1s").ShouldNot(gom.HaveOccurred())
	})
	gink.It("should run busybox", func() {
		gom.Eventually(framework.E2ERBAC(kubeClient, testNs), "20s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.CreateBusyBox(kubeClient, testNs), "20s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.CheckEndpointsCount(kubeClient, "busybox", testNs, 5, 0), "20s", "2s").ShouldNot(gom.HaveOccurred())
	})
	gink.It("should run customAnomalyDetector", func() {
		gom.Eventually(framework.CreateCustomAnamalyDetector(kubeClient, "default", testNs, labels.SelectorFromSet(map[string]string{"app": "busybox", "fail": "true"})), "20s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.CheckEndpointsCount(kubeClient, "customanomalydetector", testNs, 1, 0), "20s", "2s").ShouldNot(gom.HaveOccurred())
	})
	gink.It("should run mark 2 pods as failing", func() {
		gom.Eventually(framework.TagPod(kubeClient, testNs, labels.SelectorFromSet(map[string]string{"app": "busybox"}), 2, map[string]string{"fail": "true"}), "20s", "2s").ShouldNot(gom.HaveOccurred())
	})
})
