package cost_comparator

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/olekukonko/tablewriter"
	v1 "k8s.io/api/core/v1"
	resourcehelper "k8s.io/kubernetes/pkg/api/v1/resource"

	"github.com/gocrane/fadvisor/pkg/cache"
	"github.com/gocrane/fadvisor/pkg/consts"
	"github.com/gocrane/fadvisor/pkg/cost-comparator/config"
	"github.com/gocrane/fadvisor/pkg/cost-comparator/coster"
	"github.com/gocrane/fadvisor/pkg/util"
)

func (c *Comparator) ReportOriginalWorkloadsResourceDistribution(costerCtx *coster.CosterContext) {
	data := [][]string{}
	for kind, kindWorkloads := range costerCtx.WorkloadsSpec {
		for nn, workload := range kindWorkloads {
			var totalPrice float64 = -1
			kindWorkloadsPrices, ok := costerCtx.WorkloadsPrices[kind]
			if ok {
				if price, ok := kindWorkloadsPrices[nn]; ok {
					totalPrice = price.TotalPrice
				}
			}
			cpuReqFloat64Cores := float64(workload.Cpu.MilliValue()) / 1000.
			memReqFloat64GB := float64(workload.Mem.Value()) / consts.GB
			cpuLimFloat64Cores := float64(workload.CpuLimit.MilliValue()) / 1000.
			memLimFloat64GB := float64(workload.MemLimit.Value()) / consts.GB

			labels := workload.Workload.GetLabels()
			labelsStr := ""
			if labels != nil {
				labelsBytes, _ := json.Marshal(labels)
				labelsStr = string(labelsBytes)
			}

			// non serverless, use original pod template resource requirements
			if !workload.Serverless {
				req, lim := resourcehelper.PodRequestsAndLimits(workload.PodRef)
				reqCpu := req[v1.ResourceCPU]
				reqMem := req[v1.ResourceMemory]
				limCpu := lim[v1.ResourceCPU]
				limMem := lim[v1.ResourceMemory]
				cpuReqFloat64Cores = float64(reqCpu.MilliValue()) / 1000.
				memReqFloat64GB = float64(reqMem.Value()) / consts.GB
				cpuLimFloat64Cores = float64(limCpu.MilliValue()) / 1000.
				memLimFloat64GB = float64(limMem.Value()) / consts.GB
			}

			data = append(data,
				[]string{kind, nn.Namespace, nn.Name, Float642Str(cpuReqFloat64Cores), Float642Str(memReqFloat64GB), Float642Str(cpuLimFloat64Cores), Float642Str(memLimFloat64GB), fmt.Sprintf("%v", workload.GoodsNum), fmt.Sprintf("%v", workload.Serverless), Float642Str(totalPrice), string(workload.QoSClass), labelsStr},
			)
		}
	}

	fmt.Println("Reporting, Original Workloads Resource Distribution.....................................................................................................")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Kind", "Namespace", "Name", "CpuReq", "MemReq", "CpuLim", "MemLim", "Replicas", "Serverless", "Price", "K8SQoS", "Labels"})
		table.SetBorder(false) // Set Border to false
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
		)

		table.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		)
		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-original-workloads-distribution"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write([]string{"Kind", "Namespace", "Name", "CpuReq", "MemReq", "CpuLim", "MemLim", "Replicas", "Serverless", "Price", "K8SQoS", "Labels"})
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}
	fmt.Println()
}

func (c *Comparator) ReportRecommendedWorkloadsResourceDistribution(costerCtx *coster.CosterContext) {
	data := [][]string{}
	for kind, kindWorkloads := range costerCtx.WorkloadsRecSpec {
		for nn, workload := range kindWorkloads {
			var directTotalPrice float64 = -1
			var recTotalPrice float64 = -1
			var percentRecTotalPrice float64 = -1
			var maxRecTotalPrice float64 = -1
			var maxMarginRecTotalPrice float64 = -1
			var requestSameLimitRecPrice float64 = -1

			kindWorkloadsRecPrices, ok := costerCtx.WorkloadsRecPrices[kind]
			if ok {
				if price, ok := kindWorkloadsRecPrices[nn]; ok {
					directTotalPrice = price.DirectSpec.TotalPrice
					recTotalPrice = price.RecommendedSpec.TotalPrice
					percentRecTotalPrice = price.PercentRecommendedSpec.TotalPrice
					maxRecTotalPrice = price.MaxRecommendedSpec.TotalPrice
					maxMarginRecTotalPrice = price.MaxMarginRecommendedSpec.TotalPrice
					requestSameLimitRecPrice = price.RequestSameLimitRecommendedSpec.TotalPrice
				}
			}

			directCpuReqFloat64Cores := float64(workload.DirectSpec.Cpu.MilliValue()) / 1000.
			directMemReqFloat64GB := float64(workload.DirectSpec.Mem.Value()) / consts.GB
			directCpuLimFloat64Cores := float64(workload.DirectSpec.CpuLimit.MilliValue()) / 1000.
			directMemLimFloat64GB := float64(workload.DirectSpec.MemLimit.Value()) / consts.GB

			recCpuReqFloat64Cores := float64(workload.RecommendedSpec.Cpu.MilliValue()) / 1000.
			recMemReqFloat64GB := float64(workload.RecommendedSpec.Mem.Value()) / consts.GB
			recCpuLimFloat64Cores := float64(workload.RecommendedSpec.CpuLimit.MilliValue()) / 1000.
			recMemLimFloat64GB := float64(workload.RecommendedSpec.MemLimit.Value()) / consts.GB

			percentRecCpuReqFloat64Cores := float64(workload.PercentRecommendedSpec.Cpu.MilliValue()) / 1000.
			percentRecMemReqFloat64GB := float64(workload.PercentRecommendedSpec.Mem.Value()) / consts.GB
			percentRecCpuLimFloat64Cores := float64(workload.PercentRecommendedSpec.CpuLimit.MilliValue()) / 1000.
			percentRecMemLimFloat64GB := float64(workload.PercentRecommendedSpec.MemLimit.Value()) / consts.GB

			maxRecCpuReqFloat64Cores := float64(workload.MaxRecommendedSpec.Cpu.MilliValue()) / 1000.
			maxRecMemReqFloat64GB := float64(workload.MaxRecommendedSpec.Mem.Value()) / consts.GB
			maxRecCpuLimFloat64Cores := float64(workload.MaxRecommendedSpec.CpuLimit.MilliValue()) / 1000.
			maxRecMemLimFloat64GB := float64(workload.MaxRecommendedSpec.MemLimit.Value()) / consts.GB

			maxMarginRecCpuReqFloat64Cores := float64(workload.MaxMarginRecommendedSpec.Cpu.MilliValue()) / 1000.
			maxMarginRecMemReqFloat64GB := float64(workload.MaxMarginRecommendedSpec.Mem.Value()) / consts.GB
			maxMarginReCpuLimFloat64Cores := float64(workload.MaxMarginRecommendedSpec.CpuLimit.MilliValue()) / 1000.
			maxMarginRecMemLimFloat64GB := float64(workload.MaxMarginRecommendedSpec.MemLimit.Value()) / consts.GB

			requestSameLimitRecCpuReqFloat64Cores := float64(workload.RequestSameLimitRecommendedSpec.Cpu.MilliValue()) / 1000.
			requestSameLimitRecMemReqFloat64GB := float64(workload.RequestSameLimitRecommendedSpec.Mem.Value()) / consts.GB
			requestSameLimitRecCpuLimFloat64Cores := float64(workload.RequestSameLimitRecommendedSpec.CpuLimit.MilliValue()) / 1000.
			requestSameLimitRecMemLimFloat64GB := float64(workload.RequestSameLimitRecommendedSpec.MemLimit.Value()) / consts.GB

			containerStats, _ := json.Marshal(workload.Containers)
			recContainerStats, _ := json.Marshal(workload.RecContainers)
			recMaxContainerStats, _ := json.Marshal(workload.RecMaxContainers)
			recMaxMarginContainerStats, _ := json.Marshal(workload.RecMaxMarginContainers)
			recPercentileContainerStats, _ := json.Marshal(workload.RecPercentileContainers)
			recReqSameLimitContainerStats, _ := json.Marshal(workload.RecReqSameLimitContainers)

			data = append(data,
				[]string{kind,
					nn.Namespace,
					nn.Name,
					Float642Str(directCpuReqFloat64Cores),
					Float642Str(directMemReqFloat64GB),
					Float642Str(directCpuLimFloat64Cores),
					Float642Str(directMemLimFloat64GB),
					Float642Str(recCpuReqFloat64Cores),
					Float642Str(recMemReqFloat64GB),
					Float642Str(recCpuLimFloat64Cores),
					Float642Str(recMemLimFloat64GB),

					Float642Str(percentRecCpuReqFloat64Cores),
					Float642Str(percentRecMemReqFloat64GB),
					Float642Str(percentRecCpuLimFloat64Cores),
					Float642Str(percentRecMemLimFloat64GB),

					Float642Str(maxRecCpuReqFloat64Cores),
					Float642Str(maxRecMemReqFloat64GB),
					Float642Str(maxRecCpuLimFloat64Cores),
					Float642Str(maxRecMemLimFloat64GB),

					Float642Str(maxMarginRecCpuReqFloat64Cores),
					Float642Str(maxMarginRecMemReqFloat64GB),
					Float642Str(maxMarginReCpuLimFloat64Cores),
					Float642Str(maxMarginRecMemLimFloat64GB),

					Float642Str(requestSameLimitRecCpuReqFloat64Cores),
					Float642Str(requestSameLimitRecMemReqFloat64GB),
					Float642Str(requestSameLimitRecCpuLimFloat64Cores),
					Float642Str(requestSameLimitRecMemLimFloat64GB),

					fmt.Sprintf("%v", workload.RecommendedSpec.GoodsNum),
					fmt.Sprintf("%v", workload.RecommendedSpec.Serverless),
					Float642Str(directTotalPrice),
					Float642Str(recTotalPrice),
					Float642Str(percentRecTotalPrice),
					Float642Str(maxRecTotalPrice),
					Float642Str(maxMarginRecTotalPrice),
					Float642Str(requestSameLimitRecPrice),
					string(workload.RecommendedSpec.QoSClass),
					string(containerStats),
					string(recContainerStats),
					string(recMaxContainerStats),
					string(recMaxMarginContainerStats),
					string(recPercentileContainerStats),
					string(recReqSameLimitContainerStats),
				},
			)
		}
	}

	fmt.Println("Reporting, Recommended Workloads Resource Distribution.....................................................................................................")
	headers := []string{"Kind",
		"Namespace",
		"Name",
		"DirectCpuReq",
		"DirectMemReq",
		"DirectCpuLim",
		"DirectMemLim",
		"RecCpuReq",
		"RecMemReq",
		"RecCpuLim",
		"RecMemLim",
		"PercentRecCpuReq",
		"PercentRecMemReq",
		"PercentRecCpuLim",
		"PercentRecMemLim",
		"MaxRecCpuReq",
		"MaxRecMemReq",
		"MaxRecCpuLim",
		"MaxRecMemLim",
		"MaxMarginRecCpuReq",
		"MaxMarginRecMemReq",
		"MaxMarginReCpuLim",
		"MaxMarginRecMemLim",
		"RequestSameLimitRecCpu",
		"RequestSameLimitRecMem",
		"RequestSameLimitRecCpuLim",
		"RequestSameLimitRecMemLim",
		"Replicas",
		"Serverless",
		"DirectPrice",
		"RecTotalPrice",
		"PercentRecTotalPrice",
		"MaxRecTotalPrice",
		"MaxMarginRecTotalPrice",
		"RequestSameLimitRecPrice",
		"K8SQoS",
		"ContainerStats",
		"RecContainerStats",
		"RecMaxContainerStats",
		"RecMaxMarginContainerStats",
		"RecPercentileContainerStats",
		"RecReqSameLimitContainerStats",
	}
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader(headers)
		table.SetBorder(false) // Set Border to false
		table.SetHeaderColor(GenHeaderColor(len(headers))...)
		table.SetColumnColor(GenColumnColor(len(headers))...)
		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-recommended-workloads-distribution"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write(headers)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}

	fmt.Println()
}

func GenHeaderColor(n int) []tablewriter.Colors {
	var result []tablewriter.Colors
	for i := 0; i < n; i++ {
		result = append(result, tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor})
	}
	return result
}

func GenColumnColor(n int) []tablewriter.Colors {
	var result []tablewriter.Colors
	for i := 0; i < n; i++ {
		result = append(result, tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor})
	}
	return result
}

func (c *Comparator) ReportNodesDistribution(costerCtx *coster.CosterContext) {
	data := [][]string{}
	for nodeName, node := range costerCtx.NodesSpec {
		var totalPrice float64 = -1
		kindWorkloadsPrices, ok := costerCtx.NodesPrices[nodeName]
		if ok {
			totalPrice = kindWorkloadsPrices.TotalPrice
		}
		cpuFloat64Cores := float64(node.Cpu.MilliValue()) / 1000.
		memFloat64GB := float64(node.Mem.Value()) / consts.GB
		gpuFloat64GB := float64(node.Gpu.Value())

		nodeType := "real"
		// non serverless, use original pod template resource requirements
		if node.VirtualNode {
			totalPrice = 0
			nodeType = "vk"
		}

		data = append(data,
			[]string{nodeName, Float642Str(cpuFloat64Cores), Float642Str(memFloat64GB), Float642Str(totalPrice), nodeType, node.GpuType, Float642Str(gpuFloat64GB), node.Zone, node.Region, node.InstanceType, node.ChargeType},
		)
	}

	fmt.Println("Reporting, Nodes Resource Distribution.....................................................................................................")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Name", "Cpu", "Mem", "Price", "Type", "GpuType", "Gpu", "Zone", "Region", "InstanceType", "ChargeType"})
		table.SetBorder(false) // Set Border to false
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
		)

		table.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		)
		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-nodes-distribution"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write([]string{"Name", "Cpu", "Mem", "Price", "Type", "GpuType", "Gpu", "Zone", "Region", "InstanceType", "ChargeType"})
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}
	fmt.Println()
}

func (c *Comparator) ReportOriginalResourceSummary() {

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

	data := [][]string{
		{"clusterRequestsTotal", Float642Str(float64(clusterRequestsTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(clusterRequestsTotal.Memory().Value()) / consts.GB)},
		{"clusterLimitsTotal", Float642Str(float64(clusterLimitsTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(clusterLimitsTotal.Memory().Value()) / consts.GB)},
		{"serverfulRequestsTotal", Float642Str(float64(serverfulRequestsTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(serverfulRequestsTotal.Memory().Value()) / consts.GB)},
		{"serverfulLimitsTotal", Float642Str(float64(serverfulLimitsTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(serverfulLimitsTotal.Memory().Value()) / consts.GB)},
		{"serverlessRequestsTotal", Float642Str(float64(serverlessRequestsTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(serverlessRequestsTotal.Memory().Value()) / consts.GB)},
		{"serverlessLimitsTotal", Float642Str(float64(serverlessLimitsTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(serverlessLimitsTotal.Memory().Value()) / consts.GB)},
		{"clusterRealNodesCapacityTotal", Float642Str(float64(clusterRealNodesCapacityTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(clusterRealNodesCapacityTotal.Memory().Value()) / consts.GB)},
		{"clusterVirtualNodesCapacityTotal", clusterVirtualNodesCapacityTotal.Cpu().String(), clusterVirtualNodesCapacityTotal.Memory().String()},
	}

	fmt.Println("Reporting, Original Resource Summary.....................................................................................................")

	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Type", "Cpu", "Mem"})
		table.SetBorder(false) // Set Border to false
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor})

		table.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor})

		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-original-resource-summary"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write([]string{"Type", "Cpu", "Mem"})
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}

	fmt.Println()
}

func Float642Str(a float64) string {
	return fmt.Sprintf("%.5f", a)
}

func (c *Comparator) ReportOriginalCostSummary(costerCtx *coster.CosterContext) {
	serverfulCoster := coster.NewServerfulCoster()
	originalFee := serverfulCoster.TotalCost(costerCtx)

	data := [][]string{
		{"tke", Float642Str(originalFee.TotalCost), Float642Str(originalFee.ServerfulCost), Float642Str(originalFee.ServerlessCost), Float642Str(originalFee.ServerfulPlatformCost), Float642Str(originalFee.ServerlessPlatformCost)},
	}

	fmt.Printf("Reporting, Original Cost Summary(TimeSpan: %v, Discount: %v)............................................................................\n", c.config.TimeSpanSeconds, c.config.Discount)

	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Type", "TotalCost", "ServerfulCost", "ServerlessCost", "ServerfulPlatformCost", "ServerlessPlatformCost"})
		table.SetBorder(false) // Set Border to false
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
		)

		table.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		)

		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-original-cost-summary"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write([]string{"Type", "TotalCost", "ServerfulCost", "ServerlessCost", "ServerfulPlatformCost", "ServerlessPlatformCost"})
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}

	fmt.Println()
}

func (c *Comparator) ReportRawServerlessCostSummary(costerCtx *coster.CosterContext) {
	serverlessCoster := coster.NewServerlessCoster()
	serverlessFee := serverlessCoster.TotalCost(costerCtx)

	data := [][]string{
		{"eks", Float642Str(serverlessFee.TotalCost), Float642Str(serverlessFee.ServerfulCost), Float642Str(serverlessFee.ServerlessCost), Float642Str(serverlessFee.ServerfulPlatformCost), Float642Str(serverlessFee.ServerlessPlatformCost)},
	}

	fmt.Printf("Reporting, Direct Migrating to Serverless Cost Summary(TimeSpan: %v, Discount: %v)............................................................................\n", c.config.TimeSpanSeconds, c.config.Discount)

	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Type", "TotalCost", "ServerfulCost", "ServerlessCost", "ServerfulPlatformCost", "ServerlessPlatformCost"})
		table.SetBorder(false) // Set Border to false
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
		)

		table.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		)

		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-direct-migrate-serverless-cost-summary"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write([]string{"Type", "TotalCost", "ServerfulCost", "ServerlessCost", "ServerfulPlatformCost", "ServerlessPlatformCost"})
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}

	fmt.Println()
}

func (c *Comparator) ReportRecommendedResourceSummary(costerCtx *coster.CosterContext) {
	recomendedResourceTotal := ServerlessWorkloadsResourceTotal(costerCtx.WorkloadsRecSpec)

	data := [][]string{
		{"recomendedServerlessResourceTotal", Float642Str(float64(recomendedResourceTotal.Cpu().MilliValue()) / 1000.), Float642Str(float64(recomendedResourceTotal.Memory().Value()) / consts.GB)},
	}

	fmt.Println("Reporting, Recommended Resource Summary After Migrating to Serverless.....................................................................")

	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Type", "Cpu", "Mem"})
		table.SetBorder(false)
		// Set Border to false
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor})

		table.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor})

		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-recommended-serverless-resource-summary"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write([]string{"Type", "Cpu", "Mem"})
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}
	fmt.Println()
}

func (c *Comparator) ReportRecommendedCostSummary(costerCtx *coster.CosterContext) {
	recommendedCoster := coster.NewRecommenderCoster()
	DirectCost, RecommendedCost, PercentileCost, MaxCost, MaxMarginCost := recommendedCoster.TotalCost(costerCtx)

	data := [][]string{
		{"eks-direct-without-recommendation", Float642Str(DirectCost.TotalCost), Float642Str(DirectCost.WorkloadCost), Float642Str(DirectCost.PlatformCost)},
		{"eks-recommended-by-percentile-margin", Float642Str(RecommendedCost.TotalCost), Float642Str(RecommendedCost.WorkloadCost), Float642Str(RecommendedCost.PlatformCost)},
		{"eks-recommended-by-percentile", Float642Str(PercentileCost.TotalCost), Float642Str(PercentileCost.WorkloadCost), Float642Str(PercentileCost.PlatformCost)},
		{"eks-recommended-by-max-margin", Float642Str(MaxMarginCost.TotalCost), Float642Str(MaxMarginCost.WorkloadCost), Float642Str(MaxMarginCost.PlatformCost)},
		{"eks-recommended-by-max", Float642Str(MaxCost.TotalCost), Float642Str(MaxCost.WorkloadCost), Float642Str(MaxCost.PlatformCost)},
	}

	fmt.Printf("Reporting, Recommended Cost Summary After Migrating to Serverless(TimeSpan: %v, Discount: %v).............................................\n", c.config.TimeSpanSeconds, c.config.Discount)

	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeStdOut {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(true)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Type", "TotalCost", "WorkloadCost", "PlatformCost"})
		table.SetBorder(false) // Set Border to false
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
		)

		table.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		)

		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}

	filename := filepath.Join(c.config.DataPath, c.config.ClusterId+"-recommended-cost-summary"+".csv")
	if c.config.OutputMode == "" || c.config.OutputMode == config.OutputModeCsv {
		csvFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		csvW := csv.NewWriter(csvFile)
		csvW.Comma = '\t'
		err = csvW.Write([]string{"Type", "TotalCost", "WorkloadCost", "PlatformCost"})
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
		err = csvW.WriteAll(data)
		if err != nil {
			fmt.Println(err)
			os.Exit(255)
		}
	}

	fmt.Println()
}
