package anomalydetector

import (
	"context"
	"fmt"
	"time"

	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"

	"github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1"
)

type promDiscreteValueOutOfListAnalyser struct {
	config           v1.DiscreteValueOutOfList
	queyrAPI         promApi.API
	logger           *zap.Logger
	valueCheckerFunc func(value string) (ok bool)
}

func (p *promDiscreteValueOutOfListAnalyser) doAnalysis() (okkoByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()

	// promQL example: sum(delta(ms_rpc_count{job=\"kubernetes-pods\",run=\"foo\"}[10s])) by (code,kubernetes_pod_name)
	// p.config.PodNameKey should be "kubernetes_pod_name"
	// p.config.Key should be "code"
	m, err := p.queyrAPI.Query(ctx, p.config.PromQL, tsNow)
	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of expected type 'model.Vector': %s", err)
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

type promContinuousValueDeviationAnalyser struct {
	config   v1.ContinuousValueDeviation
	queryAPI promApi.API
	logger   *zap.Logger
}

func (p *promContinuousValueDeviationAnalyser) doAnalysis() (deviationByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()

	// promQL example: (rate(solution_price_sum{}[1m])/rate(solution_price_count{}[1m]) and delta(solution_price_count{}[1m])>70) / scalar(sum(rate(solution_price_sum{}[1m]))/sum(rate(solution_price_count{}[1m])))
	// p.config.PodNameKey should point to the label containing the pod name
	m, err := p.queryAPI.Query(ctx, p.config.PromQL, tsNow)
	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of expected type 'model.Vector': %s", err)
	}

	result := deviationByPodName{}
	for _, sample := range vector {
		metrics := sample.Metric
		podName := string(metrics[model.LabelName(p.config.PodNameKey)])
		deviation := sample.Value
		result[podName] = float64(deviation)
	}
	return result, nil
}
