package tke

import (
	"fmt"

	"k8s.io/klog"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gocrane/fadvisor/pkg/cloudsdk/qcloud/consts"
)

//https://cloud.tencent.com/document/product/457/74015
//包年包月模式
//支持调度 1C～8C 标准规格的 Pod。
//支持调度 CPU 内存比小于 1:4 的 Pod。
//CPU/核	内存区间/GiB	内存区间粒度/GiB
//1	1 - 4	1
//2	2 - 8	1
//4	8 - 16	1
//8	16 - 32	1
//
//
//按量计费模式
//支持调度 0.25C～16C 标准规格的 Pod（若为非标准规格，则自动向上转换成标准规格）。
//支持调度 CPU 内存比小于等于 1:8 的 Pod。
//
//CPU/核	内存区间/GiB	内存区间粒度/GiB
//0.25	0.5、1、2	-
//0.5	1、2、3、4	-
//1	1 - 8	1
//2	4 - 16	1
//4	8 - 32	1
//8	16 - 32	1
//12	24 - 48	1
//16	32 - 64	1

type ResourceRange struct {
	Cpu float64
	Mem []float64
}

var prePaidMatrix = []ResourceRange{
	{
		Cpu: 1,
		Mem: makeRange(1, 4),
	},
	{
		Cpu: 2,
		Mem: makeRange(2, 8),
	},
	{
		Cpu: 4,
		Mem: makeRange(8, 16),
	},
	{
		Cpu: 8,
		Mem: makeRange(16, 32),
	},
}

var postPaidMatrix = []ResourceRange{
	{
		Cpu: 0.25,
		Mem: []float64{0.5, 1, 2},
	},
	{
		Cpu: 0.5,
		Mem: makeRange(1, 4),
	},
	{
		Cpu: 1,
		Mem: makeRange(1, 8),
	},
	{
		Cpu: 2,
		Mem: makeRange(4, 16),
	},
	{
		Cpu: 4,
		Mem: makeRange(8, 32),
	},
	{
		Cpu: 8,
		Mem: makeRange(16, 32),
	},
	{
		Cpu: 12,
		Mem: makeRange(24, 48),
	},
	{
		Cpu: 16,
		Mem: makeRange(32, 64),
	},
}

func makeRange(left, right int) []float64 {
	var res []float64
	for i := left; i <= right; i++ {
		res = append(res, float64(i))
	}
	return res
}

func makeAggregates(matrix []ResourceRange) []Aggregates {
	var res []Aggregates
	for _, rr := range matrix {
		for _, mem := range rr.Mem {
			res = append(res, Aggregates{
				Cpu: rr.Cpu,
				Mem: mem,
			})
		}
	}
	return res
}

type Aggregates struct {
	Cpu float64
	Mem float64
}

var (
	// sorted by cpu first, then mem
	postPaidAggregates = makeAggregates(postPaidMatrix)
	prePaidAggregates  = makeAggregates(prePaidMatrix)
)

const (
	MaxCpuPrePaid  = 8.0
	MaxMemPrePaid  = 32.0
	MaxCpuPostPaid = 16.0
	MaxMemPostPaid = 64.0
)

// this version is implemented according to //https://cloud.tencent.com/document/product/457/74015
// used for eks super node or tke managed node to cost analyze
func Pod2EKSResourceSpecV1(param PodSpecConverterParam) (resource.Quantity, resource.Quantity, string) {
	cpu := float64(param.RawCpu.MilliValue()) / 1000
	ram := float64(param.RawMem.Value()) / 1024. / 1024. / 1024.

	resCpu := -1.0
	resMem := -1.0

	if !param.ChargeTypeForce {
		if cpu > MaxCpuPrePaid {
			param.ChargeType = consts.INSTANCECHARGETYPE_POSTPAID_BY_HOUR
		} else if cpu >= 1 && cpu <= MaxCpuPrePaid && cpu/ram > 1/4. {
			param.ChargeType = consts.INSTANCECHARGETYPE_PREPAID
		}
	}

	switch param.ChargeType {
	case consts.INSTANCECHARGETYPE_PREPAID:
		for _, item := range prePaidAggregates {
			if cpu <= item.Cpu && ram <= item.Mem {
				resCpu = item.Cpu
				resMem = item.Mem
				break
			}
		}
		// not found cpu, use max
		if resCpu < 0 {
			resCpu = MaxCpuPrePaid
		}
		if resMem < 0 {
			resMem = MaxMemPrePaid
		}
	case consts.INSTANCECHARGETYPE_POSTPAID_BY_HOUR:
		fallthrough
	default:
		for _, item := range postPaidAggregates {
			if cpu <= item.Cpu && ram <= item.Mem {
				resCpu = item.Cpu
				resMem = item.Mem
				break
			}
		}
		// not found cpu, use max
		if resCpu < 0 {
			resCpu = MaxCpuPostPaid
		}

		if resMem < 0 {
			resMem = MaxMemPostPaid
		}
	}

	return *resource.NewMilliQuantity(int64(resCpu*1000), resource.DecimalSI), *resource.NewQuantity(int64(resMem*1024*1024*1024), resource.BinarySI), param.ChargeType
}

// following version is copied from eks to keep the same logic with eks backend
var (
	MinCpu       = resource.Quantity{}
	MaxCpu       = resource.Quantity{}
	HalfCpu      = resource.Quantity{}
	OneCpu       = resource.Quantity{}
	TwoCpu       = resource.Quantity{}
	FourCpu      = resource.Quantity{}
	EightCpu     = resource.Quantity{}
	TwelveCpu    = resource.Quantity{}
	SixteenCpu   = resource.Quantity{}
	ThirtytwoCpu = resource.Quantity{}

	MaxIntelCpu = resource.Quantity{}
	MaxIntelMem = resource.Quantity{}
	MaxAmdCpu   = resource.Quantity{}
	MaxAmdMem   = resource.Quantity{}

	MinMem = resource.Quantity{}
	MaxMem = resource.Quantity{}

	OneGiMem        = resource.Quantity{}
	TwoGiMem        = resource.Quantity{}
	FourGiMem       = resource.Quantity{}
	EightGiMem      = resource.Quantity{}
	SixteenGiMem    = resource.Quantity{}
	ThirtytwoGiMem  = resource.Quantity{}
	SixtyfourGiMem  = resource.Quantity{}
	FortyeightGiMem = resource.Quantity{}
	TwentyfourGiMem = resource.Quantity{}

	mRes = make(map[resource.Quantity][]resource.Quantity)

	mIntelRes = make(map[resource.Quantity][]resource.Quantity)
	mAmdRes   = make(map[resource.Quantity][]resource.Quantity)

	v100Cpu1 = resource.Quantity{}
	v100Mem1 = resource.Quantity{}
	v100Cpu2 = resource.Quantity{}
	v100Cpu3 = resource.Quantity{}
	v100Cpu4 = resource.Quantity{}
	v100Mem4 = resource.Quantity{}

	mResV100 = make(map[resource.Quantity][]resource.Quantity)

	Nvidia2080TICpu1 = resource.Quantity{}
	Nvidia2080TIMem1 = resource.Quantity{}
	Nvidia2080TICpu2 = resource.Quantity{}
	Nvidia2080TICpu3 = resource.Quantity{}
	Nvidia2080TICpu4 = resource.Quantity{}
	Nvidia2080TIMem4 = resource.Quantity{}

	mResNvidia2080TI = make(map[resource.Quantity][]resource.Quantity)

	t4Cpu1 = resource.Quantity{}
	t4Mem1 = resource.Quantity{}
	t4Cpu2 = resource.Quantity{}
	t4Cpu3 = resource.Quantity{}
	t4Mem3 = resource.Quantity{}
	mResT4 = make(map[resource.Quantity][]resource.Quantity)

	mOnlyIntelRes = make(map[resource.Quantity][]resource.Quantity)
	mOnlyAmdRes   = make(map[resource.Quantity][]resource.Quantity)
)

func init() {
	MinCpu = resource.MustParse("250m")
	MaxCpu = resource.MustParse("32")
	HalfCpu = resource.MustParse("500m")
	OneCpu = resource.MustParse("1")
	TwoCpu = resource.MustParse("2")
	FourCpu = resource.MustParse("4")
	EightCpu = resource.MustParse("8")
	TwelveCpu = resource.MustParse("12")
	SixteenCpu = resource.MustParse("16")
	ThirtytwoCpu = resource.MustParse("32")

	MinMem = resource.MustParse("512Mi")
	MaxMem = resource.MustParse("96Gi")

	OneGiMem = resource.MustParse("1Gi")
	TwoGiMem = resource.MustParse("2Gi")
	FourGiMem = resource.MustParse("4Gi")
	EightGiMem = resource.MustParse("8Gi")
	SixteenGiMem = resource.MustParse("16Gi")
	TwentyfourGiMem = resource.MustParse("24Gi")
	ThirtytwoGiMem = resource.MustParse("32Gi")
	FortyeightGiMem = resource.MustParse("48Gi")

	mRes[MinCpu] = []resource.Quantity{MinMem, OneGiMem, TwoGiMem}
	mRes[HalfCpu] = []resource.Quantity{MinMem}
	mRes[HalfCpu] = append(mRes[HalfCpu], makeRangeArray(1, 4)...)
	mRes[OneCpu] = makeRangeArray(1, 8)
	mRes[TwoCpu] = []resource.Quantity{TwoGiMem}
	mRes[TwoCpu] = append(mRes[TwoCpu], makeRangeArray(3, 16)...)
	mRes[FourCpu] = makeRangeArray(4, 32)
	mRes[EightCpu] = makeRangeArray(8, 32)
	mRes[TwelveCpu] = makeRangeArray(24, 48)
	mRes[SixteenCpu] = makeRangeArray(16, 64)
	mRes[resource.MustParse("24")] = makeRangeArray(48, 96)
	mRes[ThirtytwoCpu] = makeRangeArray(32, 64)

	mIntelRes[MinCpu] = []resource.Quantity{MinMem, OneGiMem, TwoGiMem}
	mIntelRes[HalfCpu] = []resource.Quantity{MinMem}
	mIntelRes[HalfCpu] = append(mRes[HalfCpu], makeRangeArray(1, 4)...)
	mIntelRes[OneCpu] = makeRangeArray(1, 8)
	mIntelRes[TwoCpu] = []resource.Quantity{TwoGiMem}
	mIntelRes[TwoCpu] = append(mRes[TwoCpu], makeRangeArray(4, 16)...)
	mIntelRes[FourCpu] = makeRangeArray(8, 32)
	mIntelRes[EightCpu] = makeRangeArray(16, 32)
	mIntelRes[TwelveCpu] = makeRangeArray(24, 48)
	mIntelRes[SixteenCpu] = makeRangeArray(32, 64)

	MaxIntelCpu = resource.MustParse("24")
	MaxIntelMem = resource.MustParse("96Gi")
	mIntelRes[MaxIntelCpu] = makeRangeArray(48, 96)

	mAmdRes[OneCpu] = makeRangeArray(1, 4)
	mAmdRes[TwoCpu] = makeRangeArray(2, 8)
	mAmdRes[FourCpu] = makeRangeArray(4, 16)
	mAmdRes[EightCpu] = makeRangeArray(8, 32)
	mAmdRes[SixteenCpu] = makeRangeArray(16, 32)
	mAmdRes[ThirtytwoCpu] = makeRangeArray(32, 64)
	MaxAmdCpu = ThirtytwoCpu
	MaxAmdMem = resource.MustParse("64Gi")

	v100Cpu1 = EightCpu
	v100Mem1 = resource.MustParse("40Gi")
	v100Cpu2 = resource.MustParse("18")
	v100Cpu3 = resource.MustParse("36")
	v100Cpu4 = resource.MustParse("72")
	v100Mem4 = resource.MustParse("320Gi")

	mResV100[v100Cpu1] = []resource.Quantity{v100Mem1}
	mResV100[v100Cpu2] = []resource.Quantity{resource.MustParse("80Gi")}
	mResV100[v100Cpu3] = []resource.Quantity{resource.MustParse("160Gi")}
	mResV100[v100Cpu4] = []resource.Quantity{v100Mem4}

	Nvidia2080TICpu1 = resource.MustParse("10")
	Nvidia2080TIMem1 = resource.MustParse("40Gi")
	Nvidia2080TICpu2 = resource.MustParse("22")
	Nvidia2080TICpu3 = resource.MustParse("44")
	Nvidia2080TICpu4 = resource.MustParse("88")
	Nvidia2080TIMem4 = resource.MustParse("320Gi")

	mResNvidia2080TI[Nvidia2080TICpu1] = []resource.Quantity{Nvidia2080TIMem1}
	mResNvidia2080TI[Nvidia2080TICpu2] = []resource.Quantity{resource.MustParse("80Gi")}
	mResNvidia2080TI[Nvidia2080TICpu3] = []resource.Quantity{resource.MustParse("160Gi")}
	mResNvidia2080TI[Nvidia2080TICpu4] = []resource.Quantity{Nvidia2080TIMem4}

	t4Cpu1 = resource.MustParse("20")
	t4Mem1 = resource.MustParse("80Gi")
	t4Cpu2 = resource.MustParse("40")
	t4Cpu3 = resource.MustParse("80")
	t4Mem3 = resource.MustParse("320Gi")
	mResT4[t4Cpu1] = []resource.Quantity{t4Mem1}
	mResT4[t4Cpu2] = []resource.Quantity{resource.MustParse("160Gi")}
	mResT4[t4Cpu3] = []resource.Quantity{t4Mem3}

	mOnlyIntelRes[MinCpu] = []resource.Quantity{MinMem, OneGiMem, TwoGiMem}
	mOnlyIntelRes[HalfCpu] = []resource.Quantity{MinMem}
	mOnlyIntelRes[HalfCpu] = append(mOnlyIntelRes[HalfCpu], makeRangeArray(1, 4)...)
	mOnlyIntelRes[OneCpu] = makeRangeArray(5, 8)
	mOnlyIntelRes[TwoCpu] = makeRangeArray(9, 16)
	mOnlyIntelRes[FourCpu] = makeRangeArray(17, 32)
	mOnlyIntelRes[TwelveCpu] = makeRangeArray(24, 48)
	mOnlyIntelRes[SixteenCpu] = makeRangeArray(33, 64)
	mOnlyIntelRes[MaxIntelCpu] = makeRangeArray(48, 96)

	mOnlyAmdRes[TwoCpu] = []resource.Quantity{resource.MustParse("3Gi")}
	mOnlyAmdRes[FourCpu] = makeRangeArray(4, 7)
	mOnlyAmdRes[EightCpu] = makeRangeArray(8, 15)
	mOnlyAmdRes[SixteenCpu] = makeRangeArray(16, 31)
	mOnlyAmdRes[ThirtytwoCpu] = makeRangeArray(32, 64)
}

func makeRangeArray(min, max int) []resource.Quantity {
	numArray := []resource.Quantity{}
	for num := min; num <= max; num++ {
		numArray = append(numArray, resource.MustParse(fmt.Sprintf("%dGi", num)))
	}
	return numArray
}

func Pod2EKSResourceSpec(pod *v1.Pod, machine_type, chargeType string) (resource.Quantity, resource.Quantity) {
	cpu, mem := CalcIntelRes(pod)
	if machine_type == EKSCpuTypeValue_Intel {
		cpu, mem = CalcIntelRes(pod)
	} else if machine_type == EKSCpuTypeValue_Amd {
		cpu, mem = CalcAmdRes(pod)
	}
	return cpu, mem
}

func calcCpuMemFromResources(resReqs []v1.ResourceRequirements) (resource.Quantity, resource.Quantity) {
	cpuSumResRequest := resource.Quantity{}
	memSumResRequest := resource.Quantity{}

	cpuMaxLimit := resource.Quantity{}
	memMaxLimit := resource.Quantity{}
	for _, c := range resReqs {
		cpuSumResRequest.Add(*c.Requests.Cpu())
		memSumResRequest.Add(*c.Requests.Memory())

		tmpCpuLimit := c.Limits.Cpu()
		if tmpCpuLimit.Cmp(cpuMaxLimit) > 0 {
			cpuMaxLimit = tmpCpuLimit.DeepCopy()
		}

		tmpMemLimit := c.Limits.Memory()
		if tmpMemLimit.Cmp(memMaxLimit) > 0 {
			memMaxLimit = tmpMemLimit.DeepCopy()
		}
	}

	if cpuSumResRequest.Cmp(cpuMaxLimit) > 0 {
		cpuMaxLimit = cpuSumResRequest
	}

	if memSumResRequest.Cmp(memMaxLimit) > 0 {
		memMaxLimit = memSumResRequest
	}

	return cpuMaxLimit, memMaxLimit
}

func CalcCpuMemFromContainers(containers []v1.Container) (resource.Quantity, resource.Quantity) {
	cpuSumResRequest := resource.Quantity{}
	memSumResRequest := resource.Quantity{}

	cpuMaxLimit := resource.Quantity{}
	memMaxLimit := resource.Quantity{}
	for _, c := range containers {
		cpuSumResRequest.Add(*c.Resources.Requests.Cpu())
		memSumResRequest.Add(*c.Resources.Requests.Memory())

		tmpCpuLimit := c.Resources.Limits.Cpu()
		if tmpCpuLimit.Cmp(cpuMaxLimit) > 0 {
			cpuMaxLimit = tmpCpuLimit.DeepCopy()
		}

		tmpMemLimit := c.Resources.Limits.Memory()
		if tmpMemLimit.Cmp(memMaxLimit) > 0 {
			memMaxLimit = tmpMemLimit.DeepCopy()
		}
	}

	if cpuSumResRequest.Cmp(cpuMaxLimit) > 0 {
		cpuMaxLimit = cpuSumResRequest
	}

	if memSumResRequest.Cmp(memMaxLimit) > 0 {
		memMaxLimit = memSumResRequest
	}

	return cpuMaxLimit, memMaxLimit
}

func CalcIntelRes(pod *v1.Pod) (resource.Quantity, resource.Quantity) {
	var initCpu, initMem resource.Quantity
	if len(pod.Spec.InitContainers) != 0 {
		initCpu, initMem = CalcIntelResFromContainers(pod.Spec.InitContainers)
	}
	cpu, mem := CalcIntelResFromContainers(pod.Spec.Containers)

	klog.V(6).Infof("init container res:%s,%s", initCpu.String(), initMem.String())
	klog.V(6).Infof("container res:%s,%s", cpu.String(), mem.String())

	var resCpu, resMem resource.Quantity
	resCpu = initCpu
	if initCpu.Cmp(cpu) < 0 {
		resCpu = cpu
	}

	resMem = initMem
	if initMem.Cmp(mem) < 0 {
		resMem = mem
	}

	return CalcIntelResByMem(resCpu, resMem)
}

func CalcIntelResFromContainers(containers []v1.Container) (resource.Quantity, resource.Quantity) {
	initCpu, initMem := CalcCpuMemFromContainers(containers)
	if initCpu.IsZero() && initMem.IsZero() {
		return OneCpu, TwoGiMem
	}

	if initMem.IsZero() {
		useCpu := getMinCpu(mIntelRes, initCpu)
		return CalcIntelResByMem(useCpu, initMem)
	}

	if initCpu.Cmp(MinCpu) <= 0 {
		return CalcIntelResByMem(MinCpu, initMem)
	} else if initCpu.Cmp(HalfCpu) <= 0 {
		return CalcIntelResByMem(HalfCpu, initMem)
	} else if initCpu.Cmp(OneCpu) <= 0 {
		return CalcIntelResByMem(OneCpu, initMem)
	} else if initCpu.Cmp(TwoCpu) <= 0 {
		return CalcIntelResByMem(TwoCpu, initMem)
	} else if initCpu.Cmp(FourCpu) <= 0 {
		return CalcIntelResByMem(FourCpu, initMem)
	} else if initCpu.Cmp(EightCpu) <= 0 {
		return CalcIntelResByMem(EightCpu, initMem)
	} else if initCpu.Cmp(TwelveCpu) <= 0 {
		return CalcIntelResByMem(TwelveCpu, initMem)
	} else if initCpu.Cmp(SixteenCpu) <= 0 {
		return CalcIntelResByMem(SixteenCpu, initMem)
	} else if initCpu.Cmp(MaxIntelCpu) <= 0 {
		return CalcIntelResByMem(MaxIntelCpu, initMem)
	} else {
		return MaxIntelCpu, MaxIntelMem
	}
}

func CalcIntelResByMem(cpu, mem resource.Quantity) (resource.Quantity, resource.Quantity) {
	for _, elem := range mIntelRes[cpu] {
		if mem.Cmp(elem) <= 0 {
			return cpu, elem
		}
	}

	//not in range
	resCpu := MaxIntelCpu
	resMem := MaxIntelMem
	for key, val := range mIntelRes {
		if key.Cmp(cpu) <= 0 {
			continue
		} else {
			for _, elem := range val {
				if mem.Cmp(elem) <= 0 {
					if key.Cmp(resCpu) <= 0 {
						resCpu = key
						resMem = elem
						break
					}
				}
			}
		}
	}

	return resCpu, resMem
}

func CalcAmdRes(pod *v1.Pod) (resource.Quantity, resource.Quantity) {
	var initCpu, initMem resource.Quantity
	if len(pod.Spec.InitContainers) != 0 {
		initCpu, initMem = CalcAmdResFromContainers(pod.Spec.InitContainers)
	}
	cpu, mem := CalcAmdResFromContainers(pod.Spec.Containers)

	var resCpu, resMem resource.Quantity
	resCpu = initCpu
	if initCpu.Cmp(cpu) < 0 {
		resCpu = cpu
	}

	resMem = initMem
	if initMem.Cmp(mem) < 0 {
		resMem = mem
	}

	return CalcAmdResByMem(resCpu, resMem)
}

func CalcAmdResFromContainers(containers []v1.Container) (resource.Quantity, resource.Quantity) {
	initCpu, initMem := CalcCpuMemFromContainers(containers)
	if initCpu.IsZero() && initMem.IsZero() {
		return OneCpu, TwoGiMem
	}

	if initMem.IsZero() {
		useCpu := getMinCpu(mAmdRes, initCpu)
		return CalcAmdResByMem(useCpu, initMem)
	}

	if initCpu.Cmp(OneCpu) <= 0 {
		return CalcAmdResByMem(OneCpu, initMem)
	} else if initCpu.Cmp(TwoCpu) <= 0 {
		return CalcAmdResByMem(TwoCpu, initMem)
	} else if initCpu.Cmp(FourCpu) <= 0 {
		return CalcAmdResByMem(FourCpu, initMem)
	} else if initCpu.Cmp(EightCpu) <= 0 {
		return CalcAmdResByMem(EightCpu, initMem)
	} else if initCpu.Cmp(SixteenCpu) <= 0 {
		return CalcAmdResByMem(SixteenCpu, initMem)
	} else if initCpu.Cmp(ThirtytwoCpu) <= 0 {
		return CalcAmdResByMem(ThirtytwoCpu, initMem)
	} else {
		return MaxAmdCpu, MaxAmdMem
	}
}

func CalcAmdResByMem(cpu, mem resource.Quantity) (resource.Quantity, resource.Quantity) {
	for _, elem := range mAmdRes[cpu] {
		if mem.Cmp(elem) <= 0 {
			return cpu, elem
		}
	}

	//not in range
	resCpu := MaxAmdCpu
	resMem := MaxAmdMem
	for key, val := range mAmdRes {
		if key.Cmp(cpu) <= 0 {
			continue
		} else {
			for _, elem := range val {
				if mem.Cmp(elem) <= 0 {
					if key.Cmp(resCpu) <= 0 {
						resCpu = key
						resMem = elem
						break
					}
				}
			}
		}
	}

	return resCpu, resMem
}

func getMinCpu(res map[resource.Quantity][]resource.Quantity, cpu resource.Quantity) resource.Quantity {
	useCpu := cpu
	for key := range res {
		if useCpu.Cmp(key) <= 0 {
			useCpu = key
			break
		}
	}

	for key := range res {
		if key.Cmp(useCpu) < 0 && key.Cmp(cpu) >= 0 {
			useCpu = key
		}
	}

	return useCpu
}

func IsOnlyIntel(cpu, mem resource.Quantity) bool {
	for key, val := range mOnlyIntelRes {
		if key.Equal(cpu) {
			for _, elem := range val {
				if elem.Equal(mem) {
					return true
				}
			}
		}
	}

	return false
}

func IsOnlyAmd(cpu, mem resource.Quantity) bool {
	for key, val := range mOnlyAmdRes {
		if key.Equal(cpu) {
			for _, elem := range val {
				if elem.Equal(mem) {
					return true
				}
			}
		}
	}

	return false
}
