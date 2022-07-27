package qcloudmonitor

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"gopkg.in/gcfg.v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/fadvisor/pkg/datasource"
	"github.com/gocrane/fadvisor/pkg/metricnaming"
	_ "github.com/gocrane/fadvisor/pkg/querybuilder-providers/metricserver"
	_ "github.com/gocrane/fadvisor/pkg/querybuilder-providers/prometheus"
	_ "github.com/gocrane/fadvisor/pkg/querybuilder-providers/qcloudmonitor"
)

func TestNewProvider(t *testing.T) {
	cloudConfig, err := os.Open("/Users/tianqi/workpro/yamls/qcloud-config-sts-cls-81u5ncez.ini")
	if err != nil {
		klog.Fatalf("Couldn't open cloud provider configuration: %#v", err)
	}

	defer cloudConfig.Close()

	cfg := datasource.QCloudMonitorConfig{}
	if err := gcfg.FatalOnly(gcfg.ReadInto(&cfg, cloudConfig)); err != nil {
		klog.Errorf("Failed to read TencentCloud configuration file: %v", err)
	}
	dataSource, err := NewProvider(&cfg)
	if err != nil {
		t.Fatalf("unable to create datasource provider err: %v", err)
	}
	namespace := "kube-system"
	workloadName := "coredns"
	name := "coredns"
	cpu := metricnaming.ResourceToContainerMetricNamer(cfg.ClusterId, namespace, workloadName, name, v1.ResourceCPU)
	mem := metricnaming.ResourceToContainerMetricNamer(cfg.ClusterId, namespace, workloadName, name, v1.ResourceMemory)
	//cpuRequest := metricnaming.ContainerMetricNamer(c.config.ClusterId, kind, nn.Namespace, nn.Name, container.Name, consts.MetricCpuRequest, labels.Everything())
	//memRequest := metricnaming.ContainerMetricNamer(c.config.ClusterId, kind, nn.Namespace, nn.Name, container.Name, consts.MetricMemRequest, labels.Everything())
	//cpuLimit := metricnaming.ContainerMetricNamer(c.config.ClusterId, kind, nn.Namespace, nn.Name, container.Name, consts.MetricCpuLimit, labels.Everything())
	//memLimit := metricnaming.ContainerMetricNamer(c.config.ClusterId, kind, nn.Namespace, nn.Name, container.Name, consts.MetricMemLimit, labels.Everything())
	end := time.Now()
	step := 300 * time.Second
	start := end.Add(-24 * time.Hour)
	qRange := promapiv1.Range{
		Start: start,
		Step:  step,
		End:   end,
	}
	cpuTsList, err := dataSource.QueryTimeSeries(context.TODO(), cpu, qRange.Start, qRange.End, qRange.Step)
	if err != nil {
		t.Fatalf("Failed to query history for metric %v: %v", cpu.BuildUniqueKey(), err)
	}
	fmt.Println(cpuTsList[0])
	memTsList, err := dataSource.QueryTimeSeries(context.TODO(), mem, qRange.Start, qRange.End, qRange.Step)
	if err != nil {
		t.Fatalf("Failed to query history for metric %v: %v", mem.BuildUniqueKey(), err)
	}
	fmt.Println(memTsList[0])
}
