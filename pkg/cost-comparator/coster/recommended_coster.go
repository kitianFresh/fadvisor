package coster

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gocrane/fadvisor/pkg/cloud"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/gocrane/fadvisor/pkg/spec"
)

// eks or ask eci
type recommender struct {
}

func NewRecommenderCoster() *recommender {
	return &recommender{}
}

func (e *recommender) TotalCost(costerCtx *CosterContext) (RecommendedCost, RecommendedCost, RecommendedCost, RecommendedCost, RecommendedCost) {
	timespanInHour := float64(costerCtx.TimeSpanSeconds) / time.Hour.Seconds()

	recServerlessPodsTotalCost := 0.
	maxRecServerlessPodsTotalCost := 0.
	maxMarginServerlessPodsTotalCost := 0.
	percentServerlessPodsTotalCost := 0.
	directServerlessPodsTotalCost := 0.
	requestSameLimitServerlessPodsTotalCost := 0.

	if costerCtx.WorkloadsRecPrices == nil {
		costerCtx.WorkloadsRecPrices = make(map[string] /*kind*/ map[types.NamespacedName] /*namespace-name*/ *cloud.WorkloadRecommendedPrice)
	}

	for kind, workloadsRecSpec := range costerCtx.WorkloadsRecSpec {
		if strings.ToLower(kind) == "daemonset" {
			continue
		}
		kindWorklodsRecPrices, ok := costerCtx.WorkloadsRecPrices[kind]
		if !ok {
			kindWorklodsRecPrices = make(map[types.NamespacedName] /*namespace-name*/ *cloud.WorkloadRecommendedPrice)
			costerCtx.WorkloadsRecPrices[kind] = kindWorklodsRecPrices
		}
		for nn, workloadRecSpec := range workloadsRecSpec {
			workloadRecPrices, ok := kindWorklodsRecPrices[nn]
			if !ok {
				workloadRecPrices = &cloud.WorkloadRecommendedPrice{}
				kindWorklodsRecPrices[nn] = workloadRecPrices
			}

			recWorkloadPrice := workloadCosting(costerCtx.Pricer, timespanInHour, workloadRecSpec.RecommendedSpec, nn, kind)
			workloadCost := recWorkloadPrice * timespanInHour
			recServerlessPodsTotalCost += workloadCost
			kindWorklodsRecPrices[nn].RecommendedSpec.TotalPrice = recWorkloadPrice

			directWorkloadPrice := workloadCosting(costerCtx.Pricer, timespanInHour, workloadRecSpec.DirectSpec, nn, kind)
			directWorkloadCost := directWorkloadPrice * timespanInHour
			directServerlessPodsTotalCost += directWorkloadCost
			kindWorklodsRecPrices[nn].DirectSpec.TotalPrice = directWorkloadPrice

			recMaxWorkloadPrice := workloadCosting(costerCtx.Pricer, timespanInHour, workloadRecSpec.MaxRecommendedSpec, nn, kind)
			recMaxWorkloadCost := recMaxWorkloadPrice * timespanInHour
			maxRecServerlessPodsTotalCost += recMaxWorkloadCost
			kindWorklodsRecPrices[nn].MaxRecommendedSpec.TotalPrice = recMaxWorkloadPrice

			recMaxMarginWorkloadPrice := workloadCosting(costerCtx.Pricer, timespanInHour, workloadRecSpec.MaxMarginRecommendedSpec, nn, kind)
			recMaxMarginWorkloadCost := recMaxMarginWorkloadPrice * timespanInHour
			maxMarginServerlessPodsTotalCost += recMaxMarginWorkloadCost
			kindWorklodsRecPrices[nn].MaxMarginRecommendedSpec.TotalPrice = recMaxMarginWorkloadPrice

			percentWorkloadPrice := workloadCosting(costerCtx.Pricer, timespanInHour, workloadRecSpec.PercentRecommendedSpec, nn, kind)
			percentWorkloadPriceCost := percentWorkloadPrice * timespanInHour
			percentServerlessPodsTotalCost += percentWorkloadPriceCost
			kindWorklodsRecPrices[nn].PercentRecommendedSpec.TotalPrice = percentWorkloadPrice

			requestSameLimitWorkloadPrice := workloadCosting(costerCtx.Pricer, timespanInHour, workloadRecSpec.RequestSameLimitRecommendedSpec, nn, kind)
			requestSameLimitWorkloadPriceCost := requestSameLimitWorkloadPrice * timespanInHour
			requestSameLimitServerlessPodsTotalCost += requestSameLimitWorkloadPriceCost
			kindWorklodsRecPrices[nn].RequestSameLimitRecommendedSpec.TotalPrice = requestSameLimitWorkloadPrice
		}
	}

	platformCost := costerCtx.Pricer.PlatformPrice(cloud.PlatformParameter{Platform: cloud.ServerlessKind})

	dircetCost := RecommendedCost{
		TotalCost:    directServerlessPodsTotalCost + platformCost.TotalPrice,
		WorkloadCost: directServerlessPodsTotalCost,
		PlatformCost: platformCost.TotalPrice,
	}

	recCost := RecommendedCost{
		TotalCost:    recServerlessPodsTotalCost + platformCost.TotalPrice,
		WorkloadCost: recServerlessPodsTotalCost,
		PlatformCost: platformCost.TotalPrice,
	}

	percentCost := RecommendedCost{
		TotalCost:    percentServerlessPodsTotalCost + platformCost.TotalPrice,
		WorkloadCost: percentServerlessPodsTotalCost,
		PlatformCost: platformCost.TotalPrice,
	}

	maxRecCost := RecommendedCost{
		TotalCost:    maxRecServerlessPodsTotalCost + platformCost.TotalPrice,
		WorkloadCost: maxRecServerlessPodsTotalCost,
		PlatformCost: platformCost.TotalPrice,
	}

	maxMarginCost := RecommendedCost{
		TotalCost:    maxMarginServerlessPodsTotalCost + platformCost.TotalPrice,
		WorkloadCost: maxMarginServerlessPodsTotalCost,
		PlatformCost: platformCost.TotalPrice,
	}

	return dircetCost, recCost, percentCost, maxRecCost, maxMarginCost
}

func workloadCosting(pricer cloud.Pricer, timespanInHour float64, recommendedSpec spec.CloudPodSpec, nn types.NamespacedName, kind string) float64 {
	workloadPricing, err := pricer.ServerlessPodPrice(recommendedSpec)
	if err != nil {
		klog.Errorf("Failed to get ServerlessPodPrice for workload: %v, kind: %v, err: %v", nn, kind, err)
		return 0
	}
	var workloadPrice float64
	if workloadPricing.Cost != "" {
		workloadPrice, err = strconv.ParseFloat(workloadPricing.Cost, 64)
		if err != nil {
			klog.V(3).Infof("Could not parse pod total cost price, workload: %v, kind: %v, err: %v", nn, kind, err)
			return 0
		}
	}
	if math.IsNaN(workloadPrice) {
		klog.V(3).Infof("workloadPrice is NaN. Setting to 0. workload: %v, kind: %v", nn, kind)
		workloadPrice = 0
	}
	workloadCost := workloadPrice * timespanInHour
	return workloadCost
}
