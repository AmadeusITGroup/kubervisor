package e2e

import (
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
		gom.Eventually(framework.CreateBusyBox(kubeClient, testNs), "20s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.CheckEndpointsCount(kubeClient, "busybox", testNs, 5, 0), "20s", "2s").ShouldNot(gom.HaveOccurred())
	})
})
