package main

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amadeusitgroup/kubervisor/pkg/anomalydetector"
	v1 "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	"go.uber.org/zap"

	test "github.com/amadeusitgroup/kubervisor/test"
	kfakeclient "k8s.io/client-go/kubernetes/fake"
)

func TestSerialization(t *testing.T) {

	logger, _ := zap.NewDevelopment()
	sugar = logger.Sugar()

	pod1 := test.PodGen("pod1", "test-ns", map[string]string{"app": "test-app", labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")
	pod2 := test.PodGen("pod2", "test-ns", map[string]string{"app": "test-app", labeling.LabelTrafficKey: "yes", labeling.LabelBreakerNameKey: "foo"}, true, true, "")

	kubeClient = kfakeclient.NewSimpleClientset(pod1, pod2)
	selector = "app=test-app"
	namespace = "test-ns"

	server := httptest.NewServer(getMux())
	defer server.Close()

	cfg := anomalydetector.FactoryConfig{
		Config: anomalydetector.Config{
			BreakerStrategyConfig: v1.BreakerStrategy{
				CustomService: server.URL,
			},
		},
	}
	ad, err := anomalydetector.New(cfg)
	if err != nil {
		t.Fatalf("can't create anomaly detector %v", err)
	}
	go prepareResponse(10 * time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	pods, err := ad.GetPodsOutOfBounds()
	if err != nil {
		t.Fatalf("can't use anomaly detector %v", err)
	}

	if len(pods) != 1 {
		t.Fatalf("anomaly detector bad count. Should be 2: %v", pods)
	}

}
