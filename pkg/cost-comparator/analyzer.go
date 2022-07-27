package cost_comparator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/olekukonko/tablewriter"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/go-gota/gota/dataframe"
	"github.com/gocrane/fadvisor/pkg/cache"
	"github.com/gocrane/fadvisor/pkg/cloud"
	"github.com/gocrane/fadvisor/pkg/consts"
	"github.com/gocrane/fadvisor/pkg/cost-comparator/config"
	"github.com/gocrane/fadvisor/pkg/spec"
	"github.com/gocrane/fadvisor/pkg/util"
)

// 多维成本组合分析全部交给 jupyter，go 只做数据etl
func (c *Comparator) DoAnalysisV1() {
	podsSpec := c.GetAllPodsSpec()
	nodesSpec := c.GetAllNodesSpec()
	workloadsSpec := c.workloadsSpecCache
	workloadsContainerRecdRawData := c.GetAllWorkloadContainersRecdRawData()

	workloadInfos := MakeWorkloadInfos(workloadsSpec, workloadsContainerRecdRawData)
	podInfos := MakePodInfos(podsSpec)
	nodeInfos := MakeNodeInfos(c.baselineCloud, nodesSpec)

	workloadsDf := dataframe.LoadStructs(workloadInfos)
	nodesDf := dataframe.LoadStructs(nodeInfos)
	podsDf := dataframe.LoadStructs(podInfos)

	summaryResourceDf := c.ResourceSummary()

	//      Kind     |             Name              |   Namespace   | Replicas | OrigCpuRequest | OrigMemRequest | OrigCpuLimit | OrigMemLimit | RawRecdCpuRequest | RawRecdMemRequest | RawRecdCpuLimit | RawRecdMemLimit
	//workloadsDf.Rapply(func(series series.Series) series.Series {
	//	origCpuReq := series.Elem(4).Val().(float64)
	//	origMemReq := series.Elem(5).Val().(float64)
	//	origCpuLim := series.Elem(6).Val().(float64)
	//	origMemLim := series.Elem(7).Val().(float64)
	//	rawRecdCpuReq := series.Elem(8).Val().(float64)
	//	rawRecdMemReq := series.Elem(9).Val().(float64)
	//	//rawRecdCpuLim := series.Elem(10).Val().(float64)
	//	//rawRecdMemLim := series.Elem(11).Val().(float64)
	//
	//	directEKSOrigCpu := rawRecdCpuReq
	//	if origCpuLim != 0 {
	//		directEKSOrigCpu = origCpuLim
	//	} else if origCpuReq != 0 {
	//		directEKSOrigCpu = origCpuReq
	//	}
	//	directEKSOrigMem := rawRecdMemReq
	//	if origMemLim != 0 {
	//		directEKSOrigMem = origMemLim
	//	} else if origMemReq != 0 {
	//		directEKSOrigMem = origMemReq
	//	}
	//
	//	forcePostPaidDirectEKSSpec := c.baselineCloud.Pod2ServerlessSpecByContext(nil, cloud.PodSpecConverterParam{
	//		Enable: true,
	//		ChargeTypeForce: true,
	//		ChargeType: qcloud.INSTANCECHARGETYPE_POSTPAID_BY_HOUR,
	//		RawCpu: *resource.NewMilliQuantity(int64(directEKSOrigCpu * 1000), resource.DecimalSI),
	//		RawMem: *resource.NewQuantity(int64(directEKSOrigMem * consts.GB), resource.BinarySI),
	//	})
	//
	//	forcePrePaidDirectEKSSpec := c.baselineCloud.Pod2ServerlessSpecByContext(nil, cloud.PodSpecConverterParam{
	//		Enable: true,
	//		ChargeTypeForce: true,
	//		ChargeType: qcloud.INSTANCECHARGETYPE_PREPAID,
	//		RawCpu: *resource.NewMilliQuantity(int64(directEKSOrigCpu * 1000), resource.DecimalSI),
	//		RawMem: *resource.NewQuantity(int64(directEKSOrigMem * consts.GB), resource.BinarySI),
	//	})
	//
	//	hybridDirectEKSSpec := c.baselineCloud.Pod2ServerlessSpecByContext(nil, cloud.PodSpecConverterParam{
	//		Enable: true,
	//		ChargeTypeForce: false,
	//		ChargeType: qcloud.INSTANCECHARGETYPE_PREPAID,
	//		RawCpu: *resource.NewMilliQuantity(int64(directEKSOrigCpu * 1000), resource.DecimalSI),
	//		RawMem: *resource.NewQuantity(int64(directEKSOrigMem * consts.GB), resource.BinarySI),
	//	})
	//
	//	forcePostPaidRecdEKSSpec := c.baselineCloud.Pod2ServerlessSpecByContext(nil, cloud.PodSpecConverterParam{
	//		Enable: true,
	//		ChargeTypeForce: true,
	//		ChargeType: qcloud.INSTANCECHARGETYPE_POSTPAID_BY_HOUR,
	//		RawCpu: *resource.NewMilliQuantity(int64(rawRecdCpuReq * 1000), resource.DecimalSI),
	//		RawMem: *resource.NewQuantity(int64(rawRecdMemReq * consts.GB), resource.BinarySI),
	//	})
	//
	//	forcePrePaidRecdEKSSpec := c.baselineCloud.Pod2ServerlessSpecByContext(nil, cloud.PodSpecConverterParam{
	//		Enable: true,
	//		ChargeTypeForce: true,
	//		ChargeType: qcloud.INSTANCECHARGETYPE_PREPAID,
	//		RawCpu: *resource.NewMilliQuantity(int64(rawRecdCpuReq * 1000), resource.DecimalSI),
	//		RawMem: *resource.NewQuantity(int64(rawRecdMemReq * consts.GB), resource.BinarySI),
	//	})
	//
	//	hybridRecdEKSSpec := c.baselineCloud.Pod2ServerlessSpecByContext(nil, cloud.PodSpecConverterParam{
	//		Enable: true,
	//		ChargeTypeForce: false,
	//		ChargeType: qcloud.INSTANCECHARGETYPE_PREPAID,
	//		RawCpu: *resource.NewMilliQuantity(int64(rawRecdCpuReq * 1000), resource.DecimalSI),
	//		RawMem: *resource.NewQuantity(int64(rawRecdMemReq * consts.GB), resource.BinarySI),
	//	})
	//
	//})

	table := DataFrame2Table(summaryResourceDf)
	table.Render()

	table = DataFrame2Table(nodesDf)
	table.Render()

	table = DataFrame2Table(workloadsDf)
	table.Render()

	table = DataFrame2Table(podsDf)
	table.Render()

	summaryResourceDfCsv := filepath.Join(c.config.DataPath, c.config.ClusterId+"-original-resource-summary"+".csv")
	nodesDfCsv := filepath.Join(c.config.DataPath, c.config.ClusterId+"-nodes-distribution"+".csv")
	workloadsDfCsv := filepath.Join(c.config.DataPath, c.config.ClusterId+"-recommended-workloads-distribution"+".csv")
	podsDfCsv := filepath.Join(c.config.DataPath, c.config.ClusterId+"-pods-distribution"+".csv")

	c.WriteDataFrame(summaryResourceDf, summaryResourceDfCsv)
	c.WriteDataFrame(nodesDf, nodesDfCsv)
	c.WriteDataFrame(workloadsDf, workloadsDfCsv)
	c.WriteDataFrame(podsDf, podsDfCsv)
}

func (c *Comparator) WriteDataFrame(frame dataframe.DataFrame, csv string) {
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(csv)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = frame.WriteCSV(csvFile, dataframe.WriteHeader(true))
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}
}

func DataFrame2Table(df dataframe.DataFrame) *tablewriter.Table {
	headers := df.Names()
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeaderLine(true)
	table.SetAutoFormatHeaders(false)
	table.SetHeader(headers)
	table.SetBorder(false) // Set Border to false
	table.SetHeaderColor(GenHeaderColor(len(headers))...)
	table.SetColumnColor(GenColumnColor(len(headers))...)
	table.AppendBulk(df.Records()[1:]) // Add Bulk Data
	return table
}

type ResourceSummary struct {
	Name string
	Cpu  float64
	Mem  float64
}

func (c *Comparator) ResourceSummary() dataframe.DataFrame {
	pods := c.clusterCache.GetPods()
	pods = cache.FilterPendingFailedPods(pods)

	clusterRequestsTotal, clusterLimitsTotal := util.PodsRequestsAndLimitsTotal(pods, func(pod *v1.Pod) bool {
		return false
	}, false)

	serverfulRequestsTotal, serverfulLimitsTotal := util.PodsRequestsAndLimitsTotal(pods, c.baselineCloud.IsServerlessPod, false)
	serverlessRequestsTotal, serverlessLimitsTotal := util.PodsRequestsAndLimitsTotal(pods, c.baselineCloud.IsServerlessPod, true)

	nodes := c.clusterCache.GetNodes()
	clusterRealNodesCapacityTotal := util.NodesResourceTotal(nodes, c.baselineCloud.IsVirtualNode, false)
	clusterVirtualNodesCapacityTotal := util.NodesResourceTotal(nodes, c.baselineCloud.IsVirtualNode, true)
	datas := []ResourceSummary{
		{"clusterRequestsTotal", float64(clusterRequestsTotal.Cpu().MilliValue()) / 1000., float64(clusterRequestsTotal.Memory().Value()) / consts.GB},
		{"clusterLimitsTotal", float64(clusterLimitsTotal.Cpu().MilliValue()) / 1000., float64(clusterLimitsTotal.Memory().Value()) / consts.GB},
		{"serverfulRequestsTotal", float64(serverfulRequestsTotal.Cpu().MilliValue()) / 1000., float64(serverfulRequestsTotal.Memory().Value()) / consts.GB},
		{"serverfulLimitsTotal", float64(serverfulLimitsTotal.Cpu().MilliValue()) / 1000., float64(serverfulLimitsTotal.Memory().Value()) / consts.GB},
		{"serverlessRequestsTotal", float64(serverlessRequestsTotal.Cpu().MilliValue()) / 1000., float64(serverlessRequestsTotal.Memory().Value()) / consts.GB},
		{"serverlessLimitsTotal", float64(serverlessLimitsTotal.Cpu().MilliValue()) / 1000., float64(serverlessLimitsTotal.Memory().Value()) / consts.GB},
		{"clusterRealNodesCapacityTotal", float64(clusterRealNodesCapacityTotal.Cpu().MilliValue()) / 1000., float64(clusterRealNodesCapacityTotal.Memory().Value()) / consts.GB},
		{"clusterVirtualNodesCapacityTotal", float64(clusterVirtualNodesCapacityTotal.Cpu().MilliValue()) / 1000., float64(clusterVirtualNodesCapacityTotal.Memory().Value()) / consts.GB},
	}
	return dataframe.LoadStructs(datas)
}

func (c *Comparator) GetAllWorkloadContainersRecdRawData() map[string]map[types.NamespacedName] /*namespace-name*/ *spec.RawRecdContainers {
	results := make(map[string]map[types.NamespacedName]*spec.RawRecdContainers)
	workloads := c.workloadsSpecCache
	workloadsContainerData := c.GetWorkloadContainerData()

	for kind := range workloads {
		kindResult, ok := results[kind]
		if !ok {
			kindResult = make(map[types.NamespacedName]*spec.RawRecdContainers)
			results[kind] = kindResult
		}
		for nn, workloadPodSpec := range workloads[kind] {
			wrd := &spec.RawRecdContainers{
				Containers: make(map[string]spec.RawRecdResource),
			}

			for _, container := range workloadPodSpec.PodRef.Spec.Containers {
				cpuReqLimRatio := 1.0
				memReqLimRatio := 1.0
				if container.Resources.Requests != nil && container.Resources.Limits != nil {
					originalCpuReq, ok1 := container.Resources.Requests[v1.ResourceCPU]
					originalCpuLim, ok2 := container.Resources.Limits[v1.ResourceCPU]
					if ok1 && ok2 {
						cpuReqLimRatio = float64(originalCpuLim.MilliValue()) / float64(originalCpuReq.MilliValue())
					}

					originalMemReq, ok1 := container.Resources.Requests[v1.ResourceMemory]
					originalMemLim, ok2 := container.Resources.Limits[v1.ResourceMemory]
					if ok1 && ok2 {
						memReqLimRatio = float64(originalMemLim.MilliValue()) / float64(originalMemReq.MilliValue())
					}
				}

				kindWorkloadsContainerData, ok := workloadsContainerData[kind]
				if !ok {
					klog.Warningf("No cached data for kind %v", kind)
					continue
				}
				containersData, ok := kindWorkloadsContainerData[nn]
				if !ok {
					klog.Warningf("No cached data for kind %v, workload %v", kind, nn)
					continue
				}
				rawTsData, ok := containersData[container.Name]
				if !ok {
					klog.Warningf("No cached data for kind %v, workload %v, container %v", kind, nn, container.Name)
					continue
				}
				cpuTs := MergeTimeSeriesList(rawTsData.Cpu)
				cpuStatistics, err := c.estimator.Estimation(cpuTs, c.estimateConfig)
				if err != nil {
					klog.Errorf("Failed to estimate cpu for kind %v, workload %v, container %v, err: %v", kind, nn, container.Name, err)
					continue
				}
				memTs := MergeTimeSeriesList(rawTsData.Mem)
				memStatistics, err := c.estimator.Estimation(memTs, c.estimateConfig)
				if err != nil {
					klog.Errorf("Failed to estimate mem for kind %v, workload %v, container %v, err: %v", kind, nn, container.Name, err)
					continue
				}
				reqCpu := *cpuStatistics.Recommended
				reqMem := *memStatistics.Recommended
				wrd.Containers[container.Name] = spec.RawRecdResource{
					RawRecdCpuRequest: reqCpu,
					RawRecdMemRequest: reqMem / consts.GB,
					RawRecdCpuLimit:   reqCpu * cpuReqLimRatio,
					RawRecdMemLimit:   reqMem * memReqLimRatio / consts.GB,
				}
			}

			if klog.V(7).Enabled() {
				data, _ := jsoniter.Marshal(wrd)
				klog.V(7).Infof("Workload %v, %s", nn, wrd, string(data))
			}
			kindResult[nn] = wrd
		}
	}
	return results
}

func MakeWorkloadInfos(workloads map[string]map[types.NamespacedName]spec.CloudPodSpec, workloadRecContainers map[string]map[types.NamespacedName]*spec.RawRecdContainers) []spec.WorkloadInfo {
	var workloadInfos []spec.WorkloadInfo
	for kind, nnWorkloads := range workloads {
		for nn, workloadSpec := range nnWorkloads {

			RawRecdCpuRequest := 0.0
			RawRecdMemRequest := 0.0
			RawRecdCpuLimit := 0.0
			RawRecdMemLimit := 0.0
			kindWorkloadsRecContainers, ok := workloadRecContainers[kind]
			if ok {
				recContainers, ok := kindWorkloadsRecContainers[nn]
				if ok {
					for _, container := range recContainers.Containers {
						RawRecdCpuRequest += container.RawRecdCpuRequest
						RawRecdMemRequest += container.RawRecdMemRequest
						RawRecdCpuLimit += container.RawRecdCpuLimit
						RawRecdMemLimit += container.RawRecdMemLimit
					}
				}
			}

			replicas, err := strconv.ParseInt(fmt.Sprintf("%v", workloadSpec.GoodsNum), 10, 64)
			if err != nil {
				klog.Error(err)
			}
			klog.Infof("replicas: %v, workloadSpec.GoodsNum: %v", replicas, workloadSpec.GoodsNum)

			workloadInfos = append(workloadInfos, spec.WorkloadInfo{
				Kind:              kind,
				Namespace:         nn.Namespace,
				Name:              nn.Name,
				Replicas:          float64(replicas),
				OrigCpuRequest:    float64(workloadSpec.Cpu.MilliValue()) / 1000,
				OrigMemRequest:    float64(workloadSpec.Mem.Value()) / consts.GB,
				OrigCpuLimit:      float64(workloadSpec.CpuLimit.MilliValue()) / 1000,
				OrigMemLimit:      float64(workloadSpec.MemLimit.Value()) / consts.GB,
				RawRecdCpuRequest: RawRecdCpuRequest,
				RawRecdMemRequest: RawRecdMemRequest,
				RawRecdCpuLimit:   RawRecdCpuLimit,
				RawRecdMemLimit:   RawRecdMemLimit,
			})
		}
	}
	return workloadInfos
}

func MakeNodeInfos(price cloud.Pricer, nodes map[string] /*nodename*/ spec.CloudNodeSpec) []spec.NodeInfo {
	var nodeInfos []spec.NodeInfo
	for nodeName, nodeSpec := range nodes {
		node := nodeSpec.NodeRef
		nodeIp := ""
		CpuAllocatable := resource.Quantity{}
		MemAllocatable := resource.Quantity{}
		CpuCapacity := resource.Quantity{}
		MemCapacity := resource.Quantity{}
		if node != nil {
			// vk super node
			for _, addr := range node.Status.Addresses {
				if addr.Type == "InternalIP" {
					nodeIp = addr.Address
					break
				}
			}
			CpuAllocatable = node.Status.Allocatable[v1.ResourceCPU]
			MemAllocatable = node.Status.Allocatable[v1.ResourceMemory]
			CpuCapacity = node.Status.Capacity[v1.ResourceCPU]
			MemCapacity = node.Status.Capacity[v1.ResourceMemory]
		}
		nodePricing, err := price.NodePrice(nodeSpec)
		if err != nil {
			klog.Errorf("Failed to get node %v price: %v", nodeName, err)
			continue
		}
		var nodePrice float64
		if nodePricing.Cost != "" {
			nodePrice, err = strconv.ParseFloat(nodePricing.Cost, 64)
			if err != nil {
				klog.V(3).Infof("Could not parse total node price, node: %v", nodeName)
				continue
			}
		}
		if math.IsNaN(nodePrice) {
			klog.V(3).Infof("NodePrice is NaN. Setting to 0. node: %v, key: %v", nodeName)
			nodePrice = 0
		}

		nodeInfos = append(nodeInfos, spec.NodeInfo{
			NodeName:       nodeName,
			NodeIP:         nodeIp,
			Region:         nodeSpec.Region,
			Zone:           nodeSpec.Zone,
			Cpu:            float64(nodeSpec.Cpu.MilliValue()) / 1000,
			Mem:            float64(nodeSpec.Mem.Value()) / consts.GB,
			Gpu:            float64(nodeSpec.Gpu.Value()),
			GpuType:        nodeSpec.GpuType,
			InstanceType:   nodeSpec.InstanceType,
			ChargeType:     nodeSpec.ChargeType,
			VirtualNode:    nodeSpec.VirtualNode,
			CpuAllocatable: float64(CpuAllocatable.MilliValue()) / 1000,
			MemAllocatable: float64(MemAllocatable.Value()) / consts.GB,
			CpuCapacity:    float64(CpuCapacity.MilliValue()) / 1000,
			MemCapacity:    float64(MemCapacity.Value()) / consts.GB,
			TotalPrice:     nodePrice,
		})
	}
	return nodeInfos
}

func MakePodInfos(pods map[string] /*namespace-name*/ spec.CloudPodSpec) []spec.PodInfo {
	var podInfos []spec.PodInfo
	for nn, podSpec := range pods {
		namespace, name, err := k8scache.SplitMetaNamespaceKey(nn)
		if err != nil {
			klog.Error(err)
			continue
		}

		podInfos = append(podInfos, spec.PodInfo{
			Namespace:      namespace,
			Name:           name,
			OrigCpuRequest: float64(podSpec.Cpu.MilliValue()) / 1000,
			OrigMemRequest: float64(podSpec.Mem.Value()) / consts.GB,
			OrigCpuLimit:   float64(podSpec.CpuLimit.MilliValue()) / 1000,
			OrigMemLimit:   float64(podSpec.MemLimit.Value()) / consts.GB,
			QosClass:       string(podSpec.QoSClass),
			NodeName:       podSpec.PodRef.Spec.NodeName,
			Phase:          string(podSpec.PodRef.Status.Phase),
			Serverless:     podSpec.Serverless,
		})
	}
	return podInfos
}
