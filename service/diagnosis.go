package service

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"gopkg.in/gorp.v2"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
)

type DiagnosisService struct {
	*Service
	DB *gorp.DbMap
}

type DiagnosisTxService struct {
	*Service
	DB *gorp.Transaction
}

// 計測における全ての診断を新しい順に取得する。
func (s *DiagnosisService) ListByMeasurement(
	measurementId int,
) ([]*model.DiagnosisEntity, error) {
	if r, e := rds.ListDiagnoses(s.DB, measurementId); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		return r, nil
	}
}

// 計測内の診断を取得する。
func (s *DiagnosisService) FetchByMeasurement(
	id int,
	measurementId int,
) (*model.DiagnosisEntity, error) {
	if r, e := rds.FetchDiagnosis(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r.MeasurementId != measurementId {
		return nil, C.NewNotFoundError(
			"diagnosis_not_found",
			fmt.Sprintf("Diagnosis %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// ある病院において、指定した診断以降の既定値以上のリスクを持つ診断を全て取得する。
func (s *DiagnosisService) ListFollowings(
	hospitalId int,
	latest *int,
) ([]*model.DiagnosisEntity, error) {
	var beginTime time.Time

	if latest != nil {
		if d, e := rds.InquireDiagnosis(s.DB, *latest); e != nil {
			return nil, C.DB_OPERATION_ERROR(e)
		} else if d != nil {
			beginTime = d.RangeUntil
		} else {
			beginTime = time.Now().Add(-C.AlertBackingDuration)
		}
	} else {
		beginTime = time.Now().Add(-C.AlertBackingDuration)
	}

	if ds, e := rds.ListFollowingDiagnoses(s.DB, hospitalId, C.AlertRiskThreshold, beginTime); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		return ds, nil
	}
}

// 医師からのアクセス可否を調べる。
func (s *DiagnosisService) CheckAccessByDoctor(
	me *model.HospitalDoctor,
	id int,
	measurementId int,
) error {
	if b, e := rds.TraceDiagnosisRelations(s.DB, id, measurementId, me.Hospital.Id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if !b {
		return C.NewNotFoundError(
			"diagnosis_not_found",
			fmt.Sprintf("Diagnosis %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// 病院からのアクセス可否を調べる。
func (s *DiagnosisService) CheckAccessByHospital(
	hospitalId int,
	id int,
	measurementId int,
) error {
	if b, e := rds.TraceDiagnosisRelations(s.DB, id, measurementId, hospitalId); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if !b {
		return C.NewNotFoundError(
			"diagnosis_not_found",
			fmt.Sprintf("Diagnosis %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// 診断をまとめて登録する。
func (s *DiagnosisTxService) Register(
	entities []*model.DiagnosisEntity,
) error {
	now := time.Now()

	diagnoses := []interface{}{}

	for _, e := range entities {
		diagnoses = append(diagnoses, e.Diagnosis)
	}

	if e := s.DB.Insert(diagnoses...); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	cds := []interface{}{}

	for _, e := range entities {
		cds = append(cds, &model.ComputedDiagnosis{
			DiagnosisId: e.Diagnosis.Id,
			AlgorithmId: e.Algorithm.Id,
		})
	}

	if e := s.DB.Insert(cds...); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	contents := []interface{}{}

	for _, e := range entities {
		for _, c := range e.Contents {
			c.DiagnosisId = e.Diagnosis.Id
			contents = append(contents, c)
		}
	}

	if e := s.DB.Insert(contents...); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	for _, de := range entities {
		if e := rds.MergeNewDiagnosis(s.DB, de.Diagnosis.MeasurementId, de.Contents, now); e != nil {
			return C.DB_OPERATION_ERROR(e)
		}
	}

	return nil
}

type DiagnosisContentItem struct {
	RangeFrom  time.Time              `json:"range_from"`
	RangeUntil time.Time              `json:"range_until"`
	Parameters map[string]interface{} `json:"parameters"`
	Memo       string                 `json:"memo"`
}

// 診断を登録する。
func (s *DiagnosisTxService) RegisterByAnnotator(
	measurementId int,
	annotatorId int,
	memo string,
	items []DiagnosisContentItem,
) (*model.DiagnosisEntity, error) {
	now := time.Now()

	// ソート。
	sort.Slice(items, func(i, j int) bool {
		return items[i].RangeFrom.Before(items[j].RangeFrom)
	})

	contents := []*model.DiagnosisContent{}

	var latestBaseline *C.Baseline = nil

	// 計測における前回のベースラインを取得。
	if c, e := rds.FindLatestContent(s.DB, measurementId, items[0].RangeFrom); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if c != nil {
		if event, e := readEvents(lib.AsJson(c.Parameters)); e != nil {
			log.Printf("Unexpected parameters are found in DiagnosisContent #%d.\n", c.Id)
			log.Println(e.Error())
		} else if bl, ok := event.(*C.Baseline); ok {
			if bl.Variability != nil {
				latestBaseline = bl
			} else {
				log.Printf("DiagnosisContent #%d has Baseline-XXX parameter but does not have variability.\n", c.Id)
			}
		} else {
			log.Printf("DiagnosisContent #%d has Baseline-XXX parameter but does not represent baseline event.\n", c.Id)
		}
	}

	// 最後の基線BPM
	var baselineBpm *int = nil
	// 最大のリスク値
	var maximumRisk *int = nil

	var firstRangeFrom time.Time
	var lastRangeUntil time.Time

	for i, c := range items {
		if i > 1 && lastRangeUntil.After(c.RangeFrom) {
			return nil, C.NewBadRequestError(
				"content_overwrapped",
				fmt.Sprintf("Diagnosis contents must not be overwrapped"),
				map[string]interface{}{},
			)
		}

		params, err := json.Marshal(c.Parameters)

		if err != nil {
			return nil, C.NewBadRequestError(
				"invalid_parameter",
				err.Error(),
				map[string]interface{}{},
			)
		}

		// どの種別のイベントか確認してリスク算出。
		var risk *int = nil

		if event, e := readEvents(lib.AsJson(c.Parameters)); e != nil {
			return nil, C.NewBadRequestError(
				"invalid_ctg_event",
				e.Error(),
				map[string]interface{}{},
			)
		} else if event != nil {
			switch evt := event.(type) {
			case *C.Baseline:
				// リスクを算出し、現在の基線パラメータを更新。
				if evt.Variability != nil {
					if r := C.GetRisk(evt.Type, evt.Variability.Type, C.CTG_DecelerationNone); r >= 0 {
						risk = &r
					}
					latestBaseline = evt
				}
				baselineBpm = &evt.Value
			case *C.Deceleration:
				// リスクを算出。
				if latestBaseline != nil {
					if r := C.GetRisk(latestBaseline.Type, latestBaseline.Variability.Type, evt.Type); r >= 0 {
						risk = &r
					}
				}
			}
		}

		if risk != nil && (maximumRisk == nil || *risk > *maximumRisk) {
			maximumRisk = risk
		}

		if i == 0 {
			firstRangeFrom = c.RangeFrom
		}
		lastRangeUntil = c.RangeUntil

		contents = append(contents, &model.DiagnosisContent{
			Risk: risk,
			RangeFrom: c.RangeFrom,
			RangeUntil: c.RangeUntil,
			Parameters: model.JSON(params),
		})
	}

	diagnosis := &model.Diagnosis{
		MeasurementId: measurementId,
		BaselineBpm: baselineBpm,
		MaximumRisk: maximumRisk,
		Memo: memo,
		RangeFrom: firstRangeFrom,
		RangeUntil: lastRangeUntil,
		CreatedAt: now,
		ModifiedAt: now,
	}

	if e := s.DB.Insert(diagnosis); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	for _, c := range contents {
		c.DiagnosisId = diagnosis.Id
	}

	if len(contents) > 0 {
		holder := []interface{}{}

		for _, c := range contents {
			holder = append(holder, c)
		}

		if e := s.DB.Insert(holder...); e != nil {
			return nil, C.DB_OPERATION_ERROR(e)
		}
	}

	//ad := &model.AnnotatedDiagnosis{
	//	DiagnosisId: diagnosis.Id,
	//	AnnotatorId: annotatorId,
	//}

	//if e := s.DB.Insert(ad); e != nil {
	//	return nil, C.DB_OPERATION_ERROR(e)
	//}

	return &model.DiagnosisEntity{diagnosis, nil, contents}, nil
}

// 診断を更新する。
func (s *DiagnosisTxService) Update(
	id int,
	memo string,
) (*model.DiagnosisEntity, error) {
	var diagnosis *model.Diagnosis

	if r, e := rds.InquireDiagnosis(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"diagnosis_not_found",
			fmt.Sprintf("Diagnosis %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		diagnosis = r
	}

	diagnosis.Memo = memo
	diagnosis.ModifiedAt = time.Now()

	if _, e := s.DB.Update(diagnosis); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	if r, e := rds.FetchDiagnosis(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		return r, nil
	}
}

// 医師からの更新可否を調べる。
func (s *DiagnosisTxService) CheckUpdateByDoctor(
	me *model.HospitalDoctor,
	id int,
	measurementId int,
) error {
	if b, e := rds.TraceDiagnosisRelations(s.DB, id, measurementId, me.Hospital.Id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if !b {
		return C.NewNotFoundError(
			"diagnosis_not_found",
			fmt.Sprintf("Diagnosis %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

func readEvents(values lib.MaybeJson) (C.CTGEvent, error) {
	var baselineType *C.BaselineType = nil
	var baselineValue *int = nil

	var variabilityType *C.BaselineVariabilityType = nil
	var variabilityValue *int = nil

	var decelerationType *C.DecelerationType = nil
	var accelerationType *C.AccelerationType = nil

	for _, evt := range C.BaselineEvents {
		if iv, e := values.Get(string(evt)).AsInt64(); e == nil {
			if baselineType == nil {
				tmpEvt := evt
				baselineType = &tmpEvt
				tmpVal := int(iv)
				baselineValue = &tmpVal
			} else {
				// 基線が重複。
				return nil, fmt.Errorf("Only one baseline can be contained in an event")
			}
		}
	}

	for _, evt := range C.BaselineVariabilityEvents {
		if iv, e := values.Get(string(evt)).AsInt64(); e == nil {
			if variabilityType == nil {
				tmpEvt := evt
				variabilityType = &tmpEvt
				tmpVal := int(iv)
				variabilityValue = &tmpVal
			} else {
				// 基線細変動が重複。
				return nil, fmt.Errorf("Only one baseline variability can be contained in an event")
			}
		}
	}

	for _, evt := range C.DecelerationEvents {
		if values.Get(string(evt)).IsValid() {
			if decelerationType == nil {
				tmp := evt
				decelerationType = &tmp
			} else {
				// 基線細変動が重複。
				return nil, fmt.Errorf("Only one deceleration can be contained in an event")
			}
		}
	}

	for _, evt := range C.AccelerationEvents {
		if values.Get(string(evt)).IsValid() {
			if accelerationType == nil {
				tmp := evt
				accelerationType = &tmp
			} else {
				// 基線細変動が重複。
				return nil, fmt.Errorf("Only one acceleration can be contained in an event")
			}
		}
	}

	if baselineType != nil {
		if decelerationType != nil {
			return nil, fmt.Errorf("Baseline event can't contain deceleration")
		}
		if accelerationType != nil {
			return nil, fmt.Errorf("Baseline event can't contain acceleration")
		}

		evt := &C.Baseline{*baselineType, *baselineValue, nil}

		if variabilityType != nil {
			evt.Variability = &C.BaselineVariability{*variabilityType, *variabilityValue}
		}

		return evt, nil
	} else if decelerationType != nil {
		if accelerationType != nil {
			return nil, fmt.Errorf("Deceleration event can't contain acceleration")
		}

		return &C.Deceleration{*decelerationType}, nil
	} else if accelerationType != nil {
		return &C.Acceleration{*accelerationType}, nil
	} else {
		return nil, nil
	}
}
