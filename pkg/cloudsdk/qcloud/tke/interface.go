package tke

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

type TKE interface {
	GetEKSPodPriceByContext(pod *v1.Pod, param PodSpecConverterParam) (*QCloudPrice, error)
	GetEKSPodPrice(req *tke.GetPriceRequest) (*QCloudPrice, error)
	Pod2EKSSpecConverter
}

type PodSpecConverterParam struct {
	Enable          bool
	ChargeTypeForce bool
	ChargeType      string
	MachineType     string
	// 未规整化过的cpu mem
	RawCpu resource.Quantity
	RawMem resource.Quantity
	RawGPU resource.Quantity
}

type Pod2EKSSpecConverter interface {
	Pod2EKSSpecConverter(pod *v1.Pod) (v1.ResourceList, error)
	Pod2EKSSpecByContext(pod *v1.Pod, param PodSpecConverterParam) (v1.ResourceList, string, error)
}
