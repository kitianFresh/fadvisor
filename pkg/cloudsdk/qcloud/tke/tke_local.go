package tke

import (
	v1 "k8s.io/api/core/v1"

	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"

	"github.com/gocrane/fadvisor/pkg/cloudsdk/qcloud"
)

type TKELocalClient struct {
	config *qcloud.QCloudClientConfig
}

func NewTKELocalClient(qcc *qcloud.QCloudClientConfig) TKE {
	return &TKELocalClient{
		config: qcc,
	}
}

func (qcc *TKELocalClient) GetEKSPodPriceByContext(pod *v1.Pod, param PodSpecConverterParam) (*QCloudPrice, error) {
	panic("implement me")
}

func (qcc *TKELocalClient) Pod2EKSSpecByContext(pod *v1.Pod, param PodSpecConverterParam) (v1.ResourceList, string, error) {

	cpuQuantity, memQuantity, chargeType := Pod2EKSResourceSpecV1(param)

	return v1.ResourceList{
		v1.ResourceCPU:    cpuQuantity,
		v1.ResourceMemory: memQuantity,
	}, chargeType, nil
}

func (qcc *TKELocalClient) GetEKSPodPrice(req *tke.GetPriceRequest) (*QCloudPrice, error) {
	//cli, err := qcc.getClient()
	//if err != nil {
	//	return nil, err
	//}
	//return qcc.getEKSPodPriceWithRetry(cli, req)
	return qcc.getEKSPodPriceByLocal(req)
}

func (qcc *TKELocalClient) getEKSPodPriceByLocal(req *tke.GetPriceRequest) (*QCloudPrice, error) {
	// https://cloud.tencent.com/document/product/457/39806
	intelCpuCoreHourPrice := 0.12
	intelMemGBHourPrice := 0.05

	amdCpuCoreHourPrice := 0.055
	amdMemGBHourPrice := 0.032

	v100CpuCoreHourPrice := 0.208
	v100MemGBHourPrice := 0.122
	v100GPUCorePrice := 11.5

	t4CpuCoreHourPrice := 0.0868
	t4MemGBHourPrice := 0.0868
	t4GPUCorePrice := 5.21

	resp := &QCloudPrice{}
	//bytes, _ := json.Marshal(req)
	//klog.V(6).Infof(string(bytes))

	var goodsNum uint64 = 1

	if req.GoodsNum != nil {
		goodsNum = *req.GoodsNum
	}
	cpu := 0.
	ram := 0.
	if req.Cpu != nil {
		cpu = *req.Cpu
	}
	if req.Mem != nil {
		ram = *req.Mem
	}
	if req.Type == nil || *req.Type == EKSCpuTypeValue_Intel {
		totalCost := uint64((intelCpuCoreHourPrice*cpu+intelMemGBHourPrice*ram)*100.) * goodsNum
		resp.TotalCost = &totalCost
		return resp, nil
	}
	if *req.Type == EKSCpuTypeValue_Amd {
		totalCost := uint64((amdCpuCoreHourPrice*cpu+amdMemGBHourPrice*ram)*100) * goodsNum
		resp.TotalCost = &totalCost
		return resp, nil
	}
	if *req.Type == EKSGpuTypeValue_V100 {
		var gpu float64 = 0
		if req.Gpu != nil {
			gpu = *req.Gpu
		}
		totalCost := uint64((v100CpuCoreHourPrice*cpu+v100MemGBHourPrice*ram+v100GPUCorePrice*gpu)*100) * goodsNum
		resp.TotalCost = &totalCost
		return resp, nil
	}

	if *req.Type == EKSGpuTypeValue_T4 || *req.Type == EKSGpuTypeValue_1_4_T4 || *req.Type == EKSGpuTypeValue_1_2_T4 {
		var gpu float64 = 0
		if req.Gpu != nil {
			gpu = *req.Gpu
		}
		totalCost := uint64((t4CpuCoreHourPrice*cpu+t4MemGBHourPrice*ram+t4GPUCorePrice*gpu)*100) * goodsNum
		resp.TotalCost = &totalCost
		return resp, nil
	}

	// 当前默认使用这个
	totalCost := uint64((intelCpuCoreHourPrice*cpu+intelMemGBHourPrice*ram)*100) * goodsNum
	resp.TotalCost = &totalCost
	return resp, nil
}

func (qcc *TKELocalClient) Pod2EKSSpecConverter(pod *v1.Pod) (v1.ResourceList, error) {

	machineType := EKSPodCpuType(pod)
	if ok, gpuType := EKSPodGpuType(pod); ok {
		machineType = gpuType
	}

	cpuQuantity, memQuantity := Pod2EKSResourceSpec(pod, machineType, "")

	return v1.ResourceList{
		v1.ResourceCPU:    cpuQuantity,
		v1.ResourceMemory: memQuantity,
	}, nil
}
