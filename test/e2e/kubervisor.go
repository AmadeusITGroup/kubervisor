package e2e

import (
	"time"

	"github.com/amadeusitgroup/kubervisor/pkg/labeling"

	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"

	// for test lisibility
	gink "github.com/onsi/ginkgo"
	// for test lisibility
	gom "github.com/onsi/gomega"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/kubervisor/test/e2e/framework"
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
	gink.It("should run busybox", func() {
		gom.Eventually(framework.E2ERBAC(kubeClient, testNs), "20s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.CreateBusyBox(kubeClient, testNs), "20s", "1s").ShouldNot(gom.HaveOccurred())
	})
	gink.It("should run customAnomalyDetector", func() {
		gom.Eventually(framework.CreateCustomAnamalyDetector(kubeClient, "default", testNs, labels.SelectorFromSet(map[string]string{"app": "busybox", "fail": "true"})), "20s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.CheckEndpointsCount(kubeClient, "customanomalydetector", testNs, 1, 0), "20s", "2s").ShouldNot(gom.HaveOccurred())
	})
	gink.It("should run mark 2 pods as failing", func() {
		gom.Eventually(framework.TagPod(kubeClient, testNs, labels.SelectorFromSet(map[string]string{"app": "busybox"}), 2, map[string]string{"fail": "true"}), "20s", "2s").ShouldNot(gom.HaveOccurred())
	})
	time.Sleep(10 * time.Second)
	gink.It("should create a KubervisorService and break 2 pods", func() {
		bc := framework.NewKubervisorService("busybreak")
		bc.Spec.Service = "busybox"

		breaker := api.BreakerStrategy{}
		breaker.Name = "strategy1"
		breaker.CustomService = "customanomalydetector." + testNs
		breaker.DiscreteValueOutOfList = nil
		breaker.MinPodsAvailableCount = api.NewUInt(3)
		breaker.EvaluationPeriod = api.NewFloat64(1.0)
		bc.Spec.Breakers = []api.BreakerStrategy{breaker}

		bc.Spec.DefaultActivator.Period = api.NewFloat64(600.0)
		gom.Eventually(framework.CreateKubervisorService(kubervisorClient, bc, testNs), "10s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.IsKubervisorServiceCreated(kubervisorClient, bc.Name, testNs), "10s", "1s").ShouldNot(gom.HaveOccurred())
		time.Sleep(10 * time.Second)
		// check breaked pods looking at labels
		gom.Eventually(framework.ExpectCountPod(kubeClient, testNs, labels.SelectorFromSet(map[string]string{"app": "busybox", labeling.LabelTrafficKey: string(labeling.LabelTrafficNo)}), 2), "30s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.ExpectCountPod(kubeClient, testNs, labels.SelectorFromSet(map[string]string{"app": "busybox", labeling.LabelTrafficKey: string(labeling.LabelTrafficYes)}), 3), "10s", "1s").ShouldNot(gom.HaveOccurred())
		// check breaked pods looking at endpoints
		gom.Eventually(framework.CheckEndpointsCount(kubeClient, "busybox", testNs, 3, 0), "10s", "1s").ShouldNot(gom.HaveOccurred())
		// check the status of the CRD
		time.Sleep(5 * time.Second)
		gom.Eventually(framework.CheckKubervisorServiceStatus(kubervisorClient, "busybreak", testNs, 5, 2, 0, 0), "30s", "1s").ShouldNot(gom.HaveOccurred())
		// delete the CRD and check labels
		gom.Eventually(framework.ExpectCountPod(kubeClient, testNs, labels.SelectorFromSet(map[string]string{labeling.LabelBreakerNameKey: "busybreak"}), 5), "30s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.DeleteKubervisorService(kubervisorClient, "busybreak", testNs), "10s", "1s").ShouldNot(gom.HaveOccurred())
		gom.Eventually(framework.ExpectCountPod(kubeClient, testNs, labels.SelectorFromSet(map[string]string{labeling.LabelBreakerNameKey: "busybreak"}), 0), "30s", "1s").ShouldNot(gom.HaveOccurred())
	})

})
