package barad

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/gocrane/fadvisor/pkg/cloudsdk/qcloud"
)

const (
	_          = iota // ignore first value by assigning to blank identifier
	KB float64 = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

var regionMap = map[string]string{
	"hk": "hknew",
}

var RateLimitExceededError = errors.New("api rate limit exceeded error")

type RequestBody struct {
	Caller string `json:"caller"`
	//Dimensions          []Dimension `json:"dimensions"`
	Dimensions          interface{} `json:"dimensions"`
	EndTime             string      `json:"endTime"`
	MetricName          string      `json:"metricName"`
	Namespace           string      `json:"namespace"`
	Period              int64       `json:"period"`
	SeqID               string      `json:"seqId"`
	StartTime           string      `json:"startTime"`
	Statistics          string      `json:"statistics"`
	StorageTypePriority int64       `json:"storageTypePriority"`
	ViewName            string      `json:"viewName"`
}

type Dimension struct {
	Topic     string `json:"topic,omitempty"`
	Uin       string `json:"uin,omitempty"`
	Appid     uint64 `json:"appid,omitempty"`
	Region    string `json:"region,omitempty"`
	ClusterID string `json:"tke_cluster_instance_id,omitempty"`
	Node      string `json:"node,omitempty"`
	NodeRole  string `json:"node_role,omitempty"`
	InsId     string `json:"un_instance_id,omitempty"`
}

type ResponseBody struct {
	Code int64 `json:"code"`
	Data struct {
		Points    [][]interface{} `json:"points"`
		StartTime string          `json:"startTime"`
	} `json:"data"`
	Msg   string `json:"msg"`
	SeqID string `json:"seqId"`
}

type NodeQueryInfo struct {
	Region    string
	AppID     uint64
	ClusterID string
	NodeName  string
	InsId     string
	Start     time.Time
	End       time.Time
	Period    time.Duration
	// Cores
	CVMCpu float64
	// GB
	CVMMem float64
}

const (
	// 由于barad 只存储使用率（存储的是处理过的数据量），是个百分位数，而不是绝对量，所以这里获取峰值和均值会有一定损失，是每个时间段的百分比，再做一次处理得到当天的峰值、均值、低值
	k8s_node_cpu_usage = "k8s_node_cpu_usage"
	k8s_node_mem_usage = "k8s_node_mem_usage"

	barad_time_layout = "2006-01-02 15:04:05"

	Caller          = "tke-crane-fadvisor"
	Namespace       = "qce/tke2"
	K8sNodeViewName = "k8s_node2"
)

const (
	// Namesapce
	MetricsNamespace = "qce/tke2"
	MetricsModule    = "monitor"

	/**
	Metrics
	*/
	// workload view
	K8sWorkloadCpuCoreUsedMetric     = "k8s_workload_cpu_core_used"
	K8sWorkloadMemUsageBytesMetric   = "k8s_workload_mem_usage_bytes"
	K8sWorkloadMemNoCacheBytesMetric = "k8s_workload_mem_no_cache_bytes"
	K8sWorkloadReplicasMetric        = "k8s_workload_pod_total"
	K8sWorkloadCpuRequestsMetric     = "k8s_workload_cpu_requests"
	K8sWorkloadMemRequestsMetric     = "k8s_workload_mem_requests"
	K8sWorkloadCpuLimitsMetric       = "k8s_workload_cpu_limits"
	K8sWorkloadMemLimitsMetric       = "k8s_workload_mem_limits"

	// node view
	// 由于barad 只存储使用率（存储的是处理过的数据量），是个百分位数，而不是绝对量，所以这里获取峰值和均值会有一定损失，是每个时间段的百分比，再做一次处理得到当天的峰值、均值、低值
	K8sNodeCpuUsage = "k8s_node_cpu_usage"
	K8sNodeMemUsage = "k8s_node_mem_usage"

	// pod view
	K8sPodCpuCoreUsedMetric     = "k8s_pod_cpu_core_used"
	K8sPodMemUsageBytesMetric   = "k8s_pod_mem_usage_bytes"
	K8sPodMemNoCacheBytesMetric = "k8s_pod_mem_no_cache_bytes"

	// container view
	K8sContainerCpuCoreUsedMetric     = "k8s_container_cpu_core_used"
	K8sContainerMemUsageBytesMetric   = "k8s_container_mem_usage_bytes"
	K8sContainerMemNoCacheBytesMetric = "k8s_container_mem_no_cache_bytes"
	K8sContainerCpuCoreLimitMetric    = "k8s_container_cpu_core_limit"
	K8sContainerCpuCoreRequestMetric  = "k8s_container_cpu_core_request"
	K8sContainerMemLimitMetric        = "k8s_container_mem_limit"
	K8sContainerMemRequestMetric      = "k8s_container_mem_request"

	// Dimension
	LabelAppId         = "appid"
	LabelContainerId   = "container_id"
	LabelContainerName = "container_name"
	LabelNamespace     = "namespace"
	LabelNode          = "node"
	LabelNodeRole      = "node_role"
	LabelPodName       = "pod_name"
	LabelRegion        = "region"
	LabelClusterId     = "tke_cluster_instance_id"
	LabelUnInstanceId  = "un_instance_id"
	LabelWorkloadKind  = "workload_kind"
	LabelWorkloadName  = "workload_name"

	// View
	ViewK8sContainer = "k8s_container2"
	ViewK8sCluster   = "k8s_cluster2"
	ViewK8sComponent = "k8s_component2"
	ViewK8sNode      = "k8s_node2"
	ViewK8sPod       = "k8s_pod2"
	ViewK8sWorkload  = "k8s_workload2"
)

// getRealRegionName if found return realRegionName else return itself
func GetRealRegionName(r string) string {
	realRegion, ok := regionMap[r]
	if ok {
		return realRegion
	}
	return r
}

func GetContainerStatisticPoints(ctx context.Context, info NodeQueryInfo, statistics, metricname string) ([]float64, error) {
	return []float64{}, nil
}

func GetNodeStatisticPoints(ctx context.Context, info NodeQueryInfo, statistics, metricname string) ([]float64, error) {
	var res []float64
	startTime := info.Start.Format(barad_time_layout)
	endTime := info.End.Format(barad_time_layout)
	region := info.Region
	if regionStruct, ok := qcloud.ShortName2region[info.Region]; ok {
		region = regionStruct.Region
	}
	dimension := Dimension{
		Region:    region,
		ClusterID: info.ClusterID,
		Node:      info.NodeName,
		Appid:     info.AppID,
		NodeRole:  "Node",
		InsId:     info.InsId,
	}
	requestBody := RequestBody{
		SeqID:               string(uuid.NewUUID()),
		Caller:              Caller,
		Namespace:           Namespace,
		ViewName:            K8sNodeViewName,
		MetricName:          metricname,
		Period:              int64(info.Period.Seconds()),
		StorageTypePriority: 1,
		Statistics:          statistics,
		StartTime:           startTime,
		EndTime:             endTime,
		Dimensions:          []Dimension{dimension},
	}
	realRegion := GetRealRegionName(info.Region)
	apiUrl := fmt.Sprintf("http://%s.api.barad.tencentyun.com/metric/statisticsbatch", realRegion)

	bodyByte, err := json.Marshal(requestBody)
	if err != nil {
		return res, err
	}

	resp, err := http.Post(apiUrl, "application/json", bytes.NewReader(bodyByte))
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return res, RateLimitExceededError
	}

	var respBody ResponseBody
	respBodyByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	if err = json.Unmarshal(respBodyByte, &respBody); err != nil {
		return res, err
	}

	if respBody.Code != 0 {
		return res, fmt.Errorf("get metrics error, request: %v, resp: %s", string(bodyByte), string(respBodyByte))
	}
	if len(respBody.Data.Points) <= 0 {
		return res, nil
	}

	klog.V(3).Infof("Get metric data, request %s, resp: %s", string(bodyByte), string(respBodyByte))
	l := len(respBody.Data.Points[0])
	if l <= 0 {
		return res, nil
	}

	for i := 0; i < l; i++ {
		dataPointX := respBody.Data.Points[0][i]
		if dataPointX != nil {
			value, ok := dataPointX.(float64)
			if ok {
				res = append(res, value)
			}
		}
	}
	return res, nil
}
