package cloud

import (
	"github.com/gocrane/fadvisor/pkg/spec"
	"k8s.io/apimachinery/pkg/api/resource"
)

//? cost means [price * timespan]. maybe we refine the price and cost meaning later, now the price and cost is same
type Prices struct {
	TotalPrice    float64
	DiscountPrice *float64
}

type WorkloadRecommendedPrice struct {
	DirectSpec                      Prices
	RecommendedSpec                 Prices
	PercentRecommendedSpec          Prices
	MaxRecommendedSpec              Prices
	MaxMarginRecommendedSpec        Prices
	RequestSameLimitRecommendedSpec Prices
}

type Pricer interface {
	NodePricer
	PodPricer
	PlatformPricer
}

type NodePricer interface {
	NodePrice(spec spec.CloudNodeSpec) (*Node, error)
}

type ResourceParam struct {
	Cpu, Ram, Gpu resource.Quantity
	ChargeType    string
	MachineType   string
}

type PodPricer interface {
	// ServerlessPodPrice means this is a serverless pod instance, such as TencentCloud EKS pod, or AliCloud ECI
	ServerlessPodPriceByContext(param ResourceParam) (*Pod, error)
	// ServerlessPodPrice means this is a serverless pod instance, such as TencentCloud EKS pod, or AliCloud ECI
	ServerlessPodPrice(spec spec.CloudPodSpec) (*Pod, error)
	// PodPrice means this pod is in the non-serverless real node. the node is not virtual kubelet. not used now
	PodPrice(spec spec.CloudPodSpec) (*Pod, error)
}

type PlatformKind string

const (
	ServerlessKind PlatformKind = "serverless"
	ServerfulKind  PlatformKind = "serverful"
)

type PlatformParameter struct {
	// cluster nodes number
	Nodes        *int32
	ClusterLevel *string
	Platform     PlatformKind
}

type PlatformPricer interface {
	PlatformPrice(cp PlatformParameter) *Prices
}
