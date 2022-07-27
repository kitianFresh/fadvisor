package barad

import (
	"github.com/gocrane/fadvisor/pkg/metricquery"
	"github.com/gocrane/fadvisor/pkg/querybuilder"
)

var _ querybuilder.Builder = &builder{}

type builder struct {
	metric *metricquery.Metric
}

func NewBaradQueryBuilder(metric *metricquery.Metric) querybuilder.Builder {
	return &builder{
		metric: metric,
	}
}

func (b builder) BuildQuery(behavior querybuilder.BuildQueryBehavior) (*metricquery.Query, error) {
	return metricServerQuery(&metricquery.GenericQuery{Metric: b.metric}), nil
}

func metricServerQuery(query *metricquery.GenericQuery) *metricquery.Query {
	return &metricquery.Query{
		Type:         metricquery.BaradMetricSource,
		GenericQuery: query,
	}
}

func init() {
	querybuilder.RegisterBuilderFactory(metricquery.BaradMetricSource, NewBaradQueryBuilder)
}
