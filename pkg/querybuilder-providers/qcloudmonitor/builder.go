package qcloudmonitor

import (
	"github.com/gocrane/fadvisor/pkg/metricquery"
	"github.com/gocrane/fadvisor/pkg/querybuilder"
)

var _ querybuilder.Builder = &builder{}

type builder struct {
	metric *metricquery.Metric
}

func NewQCloudMonitorQueryBuilder(metric *metricquery.Metric) querybuilder.Builder {
	return &builder{
		metric: metric,
	}
}

func (b builder) BuildQuery(behavior querybuilder.BuildQueryBehavior) (*metricquery.Query, error) {
	return qcloudMonitorQuery(&metricquery.GenericQuery{Metric: b.metric}), nil
}

func qcloudMonitorQuery(query *metricquery.GenericQuery) *metricquery.Query {
	return &metricquery.Query{
		Type:         metricquery.QCloudMonitorMetricSource,
		GenericQuery: query,
	}
}

func init() {
	querybuilder.RegisterBuilderFactory(metricquery.QCloudMonitorMetricSource, NewQCloudMonitorQueryBuilder)
}
