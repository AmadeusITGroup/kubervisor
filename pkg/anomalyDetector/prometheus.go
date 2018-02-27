package anomalyDetector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	promClient "github.com/prometheus/client_golang/api"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type promDiscreteValueOutOfListAnalyser struct {
	config           v1.DiscreteValueOutOfList
	prometheusClient promClient.Client
	logger           *zap.Logger
	valueCheckerFunc func(value string) (ok bool)
}

func (p *promDiscreteValueOutOfListAnalyser) doAnalysis() (okkoByPodName, error) {

	ctx := context.Background()
	qAPI := promApi.NewAPI(p.prometheusClient)
	tsNow := time.Now()

	// promQL example: sum(delta(ms_rpc_count{job=\"kubernetes-pods\",run=\"foo\"}[10s])) by (code,kubernetes_pod_name)
	// p.config.PodNameKey should be "kubernetes_pod_name"
	// p.config.Key should be "code"
	m, err := qAPI.Query(ctx, p.config.PromQL, tsNow)
	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of 'model.Vector'", err)
	}

	return p.buildCounters(vector), nil
}

func (p *promDiscreteValueOutOfListAnalyser) buildCounters(vector model.Vector) okkoByPodName {
	countersByPods := okkoByPodName{}

	for _, sample := range vector {
		metrics := sample.Metric
		podName := string(metrics[model.LabelName(p.config.PodNameKey)])
		counters := countersByPods[podName]

		discreteValue := metrics[model.LabelName(p.config.Key)]
		if p.valueCheckerFunc(string(discreteValue)) {
			counters.ok += uint(sample.Value)
		} else {
			counters.ko += uint(sample.Value)
		}
		countersByPods[podName] = counters
	}
	return countersByPods
}
