package spec

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type PodFilter interface {
	// IsServerlessPod return if the pod is serverless pod, which means this pod is in virtual kubelet. It is controlled by kubernetes control panel.
	// for example, TencentCloud eks pod, or AliCloud eci pod
	IsServerlessPod(pod *v1.Pod) bool
}

type NodeFilter interface {
	// IsVirtualNode return the node is running virtual kubelet
	// for example, TencentCloud eks eklet, or AliCloud virtual kubelet
	IsVirtualNode(node *v1.Node) bool
}

type CloudPodSpec struct {
	// only pod template is useful, ignore name, now we have not completely know the struct like
	PodRef   *v1.Pod
	Workload *unstructured.Unstructured
	Cpu      resource.Quantity
	Mem      resource.Quantity
	CpuLimit resource.Quantity
	MemLimit resource.Quantity
	Zone     string
	// v100，t4，amd，Default：intel
	MachineArch   string
	Gpu           resource.Quantity
	PodChargeType string
	// time span
	TimeSpan uint64
	// replicas, for pod value is 1, for workload, value is replicas of the workload spec
	GoodsNum uint64
	// serverless pod or not
	Serverless bool

	QoSClass v1.PodQOSClass
}

type CloudNodeSpec struct {
	NodeRef      *v1.Node
	Cpu          resource.Quantity
	Mem          resource.Quantity
	Gpu          resource.Quantity
	GpuType      string
	InstanceType string
	ChargeType   string
	Zone         string
	Region       string
	// virtual node or not
	VirtualNode bool
}

type PodRawRecommendData struct {
	WorkloadKind string
	WorkloadName string
	Namespace    string
	Name         string
	NodeName     string

	OrigCpuRequest float64
	OrigMemRequest float64
	OrigCpuLimit   float64
	OrigMemLimit   float64

	RawRecdCpuRequest float64
	RawRecdMemRequest float64
	RawRecdCpuLimit   float64
	RawRecdMemLimit   float64
}

type K8sObjectInfo struct {
	WorkloadInfos []*WorkloadInfo
	PodInfos      []*PodInfo
	NodeInfos     []*NodeInfo
}

type PodInfo struct {
	Namespace          string
	Name               string
	WorkloadName       string
	WorkloadKind       string
	WorkloadAPIVersion string

	OrigCpuRequest float64
	OrigMemRequest float64
	OrigCpuLimit   float64
	OrigMemLimit   float64

	RawRecdCpuRequest float64
	RawRecdMemRequest float64
	RawRecdCpuLimit   float64
	RawRecdMemLimit   float64

	QosClass     string
	NodeName     string
	NodeInstance string
	NodeIP       string
	Reason       string
	Phase        string
	Serverless   bool
}

type WorkloadInfo struct {
	Kind           string
	Name           string
	Namespace      string
	Replicas       float64
	OrigCpuRequest float64
	OrigMemRequest float64
	OrigCpuLimit   float64
	OrigMemLimit   float64

	RawRecdCpuRequest float64
	RawRecdMemRequest float64
	RawRecdCpuLimit   float64
	RawRecdMemLimit   float64
}

type RawRecdResource struct {
	RawRecdCpuRequest float64
	RawRecdMemRequest float64
	RawRecdCpuLimit   float64
	RawRecdMemLimit   float64
}

type RawRecdContainers struct {
	Containers map[string]RawRecdResource
}

type NodeInfo struct {
	NodeName       string
	NodeInstance   string
	NodeIP         string
	CpuAllocatable float64
	MemAllocatable float64
	CpuCapacity    float64
	MemCapacity    float64
	Cpu            float64
	Mem            float64
	Gpu            float64
	GpuType        string
	InstanceType   string
	ChargeType     string
	Zone           string
	Region         string
	// virtual node or not
	VirtualNode bool
	TotalPrice  float64
}

type WorkloadRecommendedData struct {
	DirectSpec                      CloudPodSpec
	RecommendedSpec                 CloudPodSpec
	PercentRecommendedSpec          CloudPodSpec
	MaxRecommendedSpec              CloudPodSpec
	MaxMarginRecommendedSpec        CloudPodSpec
	RequestSameLimitRecommendedSpec CloudPodSpec
	Containers                      map[string]*ContainerRecommendedData
	RecContainers                   map[string]*RecContainerData
	RecMaxContainers                map[string]*RecContainerData
	RecMaxMarginContainers          map[string]*RecContainerData
	RecPercentileContainers         map[string]*RecContainerData
	RecReqSameLimitContainers       map[string]*RecContainerData
}

type Statistic struct {
	Percentile     *float64
	Max            *float64
	MaxRecommended *float64
	Recommended    *float64
}

type RecContainerData struct {
	ContainerName        string
	CpuReq               float64
	MemReq               float64
	CpuLim               float64
	MemLim               float64
	ResourceRequirements v1.ResourceRequirements
}

type ContainerRecommendedData struct {
	Cpu *Statistic
	Mem *Statistic
}
