package constant

// 診断記録。
type CTGEventType string
type BaselineType CTGEventType
type BaselineVariabilityType CTGEventType
type DecelerationType CTGEventType
type AccelerationType CTGEventType

const (
	CTG_BaselineNormal BaselineType = "Baseline-NORMAL"
	CTG_BaselineAcceleration BaselineType = "Baseline-ACCELERATION"
	CTG_BaselineDeceleration BaselineType = "Baseline-DECELERATION"
	CTG_BaselineHiDeceleration BaselineType = "Baseline-HiDECELERATION"

	CTG_BaselineVariabilityNormal BaselineVariabilityType = "BaselineVariability-NORMAL"
	CTG_BaselineVariabilityDecrease BaselineVariabilityType = "BaselineVariability-DECREASE"
	CTG_BaselineVariabilityIncrease BaselineVariabilityType = "BaselineVariability-INCREASE"
	CTG_BaselineVariabilityLost BaselineVariabilityType = "BaselineVariability-LOST"
	CTG_BaselineVariabilitySinusoidal BaselineVariabilityType = "BaselineVariability-SINUSOIDAL"

	CTG_DecelerationNone DecelerationType = "Deceleration-None"
	CTG_DecelerationED DecelerationType = "Deceleration-ED"
	CTG_DecelerationLowLD DecelerationType = "Deceleration-LOW_LD"
	CTG_DecelerationHiLD DecelerationType = "Deceleration-HI_LD"
	CTG_DecelerationLowPD DecelerationType = "Deceleration-LOW_PD"
	CTG_DecelerationHiPD DecelerationType = "Deceleration-HI_PD"
	CTG_DecelerationLowVD DecelerationType = "Deceleration-LOW_VD"
	CTG_DecelerationHiVD DecelerationType = "Deceleration-HI_VD"

	CTG_Acceleration AccelerationType = "Acceleration"
)

var (
	BaselineEvents = []BaselineType{
		CTG_BaselineNormal,
		CTG_BaselineAcceleration,
		CTG_BaselineDeceleration,
		CTG_BaselineHiDeceleration,
	}

	BaselineVariabilityEvents = []BaselineVariabilityType{
		CTG_BaselineVariabilityNormal,
		CTG_BaselineVariabilityDecrease,
		CTG_BaselineVariabilityIncrease,
		CTG_BaselineVariabilityLost,
		CTG_BaselineVariabilitySinusoidal,
	}

	DecelerationEvents = []DecelerationType{
		CTG_DecelerationNone,
		CTG_DecelerationED,
		CTG_DecelerationLowLD,
		CTG_DecelerationHiLD,
		CTG_DecelerationLowPD,
		CTG_DecelerationHiPD,
		CTG_DecelerationLowVD,
		CTG_DecelerationHiVD,
	}

	AccelerationEvents = []AccelerationType{
		CTG_Acceleration,
	}

	VariabilityNormalRisks = [][]int{
		[]int{  1,  2,  2,  3,  3,  3,  3,  4 },
		[]int{  2,  2,  3,  3,  3,  4,  3,  4 },
		[]int{  3,  3,  3,  4,  4,  4,  4,  4 },
		[]int{  4,  4, -1,  4,  4,  4, -1, -1 },
	}

	VariabilityDecreaseRisks = [][]int{
		[]int{  2,  3,  3,  4,  3,  4,  4,  5 },
		[]int{  3,  3,  4,  4,  4,  5,  4,  5 },
		[]int{  4,  4,  4,  5,  5,  5,  5,  5 },
		[]int{  5,  5, -1,  5,  5,  5, -1, -1 },
	}

	VariabilityLostRisks = []int{
		4,  5,  5,  5,  5,  5,  5,  5,
	}

	VariabilityIncreaseRisks = []int{
		2,  2,  3,  3,  3,  4,  3,  4,
	}

	VariabilitySinusoidalRisks = []int{
		4,  4,  4,  4,  5,  5,  5,  5,
	}
)

type CTGEvent interface {
	EventType() CTGEventType
}

type Baseline struct {
	Type        BaselineType
	Value       int
	Variability *BaselineVariability
}

func (e Baseline) EventType() CTGEventType {
	return CTGEventType(e.Type)
}

type BaselineVariability struct {
	Type  BaselineVariabilityType
	Value int
}

func (e BaselineVariability) EventType() CTGEventType {
	return CTGEventType(e.Type)
}

type Deceleration struct {
	Type DecelerationType
}

func (e Deceleration) EventType() CTGEventType {
	return CTGEventType(e.Type)
}

type Acceleration struct {
	Type AccelerationType
}

func (e Acceleration) EventType() CTGEventType {
	return CTGEventType(e.Type)
}

func GetRisk(baseline BaselineType, variability BaselineVariabilityType, deceleration DecelerationType) int {
	bi := -1

	for i, evt := range BaselineEvents {
		if evt == baseline {
			bi = i
			break
		}
	}

	di := -1

	for i, evt := range DecelerationEvents {
		if evt == deceleration {
			di = i
			break
		}
	}

	if bi < 0 || di < 0 {
		return -1
	}

	switch variability {
	case CTG_BaselineVariabilityNormal:
		return VariabilityNormalRisks[bi][di]
	case CTG_BaselineVariabilityDecrease:
		return VariabilityDecreaseRisks[bi][di]
	case CTG_BaselineVariabilityLost:
		return VariabilityLostRisks[di]
	case CTG_BaselineVariabilityIncrease:
		return VariabilityIncreaseRisks[di]
	case CTG_BaselineVariabilitySinusoidal:
		return VariabilitySinusoidalRisks[di]
	default:
		return -1
	}
}