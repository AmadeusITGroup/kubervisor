package e2e

import (
	api "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"

	// for test lisibility
	. "github.com/onsi/ginkgo"
	// for test lisibility
	. "github.com/onsi/gomega"

	"github.com/amadeusitgroup/podkubervisor/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/podkubervisor/test/e2e/framework"
)

var kubervisorClient versioned.Interface
var kubeClient clientset.Interface

const testNs = api.NamespaceDefault

var _ = BeforeSuite(func() {
	kubervisorClient, kubeClient = framework.BuildAndSetClients()
})

var _ = AfterSuite(func() {

})

var _ = Describe("BreakerConfig CRUD", func() {
	It("should create a BreakerConfig", func() {
		bc := framework.NewBreakerConfig("foo")

		Eventually(framework.CreateBreakerConfig(kubervisorClient, bc, testNs), "5s", "1s").ShouldNot(HaveOccurred())
		Eventually(framework.IsBreakerConfigCreated(kubervisorClient, bc.Name, testNs), "5s", "1s").ShouldNot(HaveOccurred())

	})
})
