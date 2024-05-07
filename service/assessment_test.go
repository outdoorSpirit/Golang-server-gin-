package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestServiceAssessment_Execute(t *testing.T) {
	db := lib.GetDB(lib.WriteDBKey)

	F.Truncate(db, "diagnosis_algorithm")

	F.Insert(db, model.DiagnosisAlgorithm{}, 0, 3, func(i int, r F.Record) {
		r["Name"] = fmt.Sprintf("alg-%04d", i)
		r["Version"] = "1.0.0"
	})

	// 計測はID以外利用しない。計測データは出力に必要。前回の診断は、nilでなければ利用される。
	entity := &model.DiagnosisMeasurmentEntity{
		Measurement: &model.Measurement{
			Id: 3,
		},
		LatestDiagnosis: nil,
		HeartRates: []*model.HeartRate{},
		TOCOs: []*model.TOCO{},
	}

	beginTime := time.Date(2021, time.January, 2, 3, 4, 5, 0, time.UTC)

	for i := 0; i < 10; i++ {
		entity.HeartRates = append(entity.HeartRates, &model.HeartRate{
			3, "", "", 100+i, beginTime.Add(time.Duration(i)*time.Minute),
		})
	}
	for i := 0; i < 20; i++ {
		entity.TOCOs = append(entity.TOCOs, &model.TOCO{
			3, "", "", 200+i, beginTime.Add(time.Duration(i*2)*time.Minute),
		})
	}

	// 診断日時、データ範囲は計算には利用されないため、適当な値にしておく。
	duration := time.Duration(10)*time.Minute
	diagnosisTime := beginTime.Add(duration)

	// テスト用のディレクトリを指定。
	rootDir := filepath.Join(os.Getenv("SERVER_ROOT"), "./data/test/ctg-assessment")
	command := "CTGRiskAssessmentor"
	parametersFile := "parameter.txt"

	s := &AssessmentService{nil, lib.GetDB(lib.ReadDBKey)}

	diagnosis, err := s.Execute(entity, diagnosisTime, duration, rootDir, command, parametersFile, "alg-0002", "1.0.0")

	assert.NoError(t, err)

	assert.EqualValues(t, 3, diagnosis.MeasurementId)
	assert.EqualValues(t, 11, *diagnosis.BaselineBpm)
	assert.EqualValues(t, 262, *diagnosis.MaximumRisk)
	assert.EqualValues(t, 6, len(diagnosis.Contents))

	parameters := map[string]interface{}{}

	assert.EqualValues(t, 1235089854980*1000000, diagnosis.Contents[0].RangeFrom.UnixNano())
	assert.EqualValues(t, 1235089947730*1000000, diagnosis.Contents[0].RangeUntil.UnixNano())
	assert.Nil(t, diagnosis.Contents[0].Risk)
	assert.NoError(t, json.Unmarshal(diagnosis.Contents[0].Parameters, &parameters))
	assert.EqualValues(t, map[string]interface{}{
		"Acceleration": nil,
		"BpmArg": "null",
	}, parameters)

	parameters = map[string]interface{}{}

	assert.EqualValues(t, 1235089975730*1000000, diagnosis.Contents[1].RangeFrom.UnixNano())
	assert.EqualValues(t, 1235090018730*1000000, diagnosis.Contents[1].RangeUntil.UnixNano())
	assert.Nil(t, diagnosis.Contents[1].Risk)
	assert.NoError(t, json.Unmarshal(diagnosis.Contents[1].Parameters, &parameters))
	assert.EqualValues(t, map[string]interface{}{
		"Acceleration": nil,
	}, parameters)

	parameters = map[string]interface{}{}

	assert.EqualValues(t, 1235090042230*1000000, diagnosis.Contents[2].RangeFrom.UnixNano())
	assert.EqualValues(t, 1235090085730*1000000, diagnosis.Contents[2].RangeUntil.UnixNano())
	assert.Nil(t, diagnosis.Contents[2].Risk)
	assert.NoError(t, json.Unmarshal(diagnosis.Contents[2].Parameters, &parameters))
	assert.EqualValues(t, map[string]interface{}{
		"Acceleration": nil,
	}, parameters)

	parameters = map[string]interface{}{}

	assert.EqualValues(t, 1235090139730*1000000, diagnosis.Contents[3].RangeFrom.UnixNano())
	assert.EqualValues(t, 1235090276980*1000000, diagnosis.Contents[3].RangeUntil.UnixNano())
	assert.EqualValues(t, 100, *diagnosis.Contents[3].Risk)
	assert.NoError(t, json.Unmarshal(diagnosis.Contents[3].Parameters, &parameters))
	assert.EqualValues(t, map[string]interface{}{
		"Baseline-NORMAL": float64(100),
		"BaselineVariavility-INCREASE": float64(20),
		"Risk": float64(100),
	}, parameters)

	parameters = map[string]interface{}{}

	assert.EqualValues(t, 1235090300000*1000000, diagnosis.Contents[4].RangeFrom.UnixNano())
	assert.EqualValues(t, 1235093300000*1000000, diagnosis.Contents[4].RangeUntil.UnixNano())
	assert.EqualValues(t, 262, *diagnosis.Contents[4].Risk)
	assert.NoError(t, json.Unmarshal(diagnosis.Contents[4].Parameters, &parameters))
	assert.EqualValues(t, map[string]interface{}{
		"Baseline-NORMAL": float64(21),
		"Risk": float64(262),
	}, parameters)

	parameters = map[string]interface{}{}

	assert.EqualValues(t, 1235090400000*1000000, diagnosis.Contents[5].RangeFrom.UnixNano())
	assert.EqualValues(t, 1235094400000*1000000, diagnosis.Contents[5].RangeUntil.UnixNano())
	assert.EqualValues(t, 200, *diagnosis.Contents[5].Risk)
	assert.NoError(t, json.Unmarshal(diagnosis.Contents[5].Parameters, &parameters))
	assert.EqualValues(t, map[string]interface{}{
		"Baseline-NORMAL": float64(11),
		"Risk": float64(200),
	}, parameters)
}

func TestServiceAssessment_ExecuteWithBpm(t *testing.T) {
	db := lib.GetDB(lib.WriteDBKey)

	F.Truncate(db, "diagnosis_algorithm")

	F.Insert(db, model.DiagnosisAlgorithm{}, 0, 3, func(i int, r F.Record) {
		r["Name"] = fmt.Sprintf("alg-%04d", i)
		r["Version"] = "1.0.0"
	})

	// 計測はID以外利用しない。計測データは出力に必要。前回の診断は、nilでなければ利用される。
	bpm := 150
	entity := &model.DiagnosisMeasurmentEntity{
		Measurement: &model.Measurement{
			Id: 3,
		},
		LatestDiagnosis: &model.Diagnosis{
			BaselineBpm: &bpm,
		},
		HeartRates: []*model.HeartRate{},
		TOCOs: []*model.TOCO{},
	}

	beginTime := time.Date(2021, time.January, 2, 3, 4, 5, 0, time.UTC)

	for i := 0; i < 10; i++ {
		entity.HeartRates = append(entity.HeartRates, &model.HeartRate{
			3, "", "", 100+i, beginTime.Add(time.Duration(i)*time.Minute),
		})
	}
	for i := 0; i < 20; i++ {
		entity.TOCOs = append(entity.TOCOs, &model.TOCO{
			3, "", "", 200+i, beginTime.Add(time.Duration(i*2)*time.Minute),
		})
	}

	// 診断日時、データ範囲は計算には利用されないため、適当な値にしておく。
	duration := time.Duration(10)*time.Minute
	diagnosisTime := beginTime.Add(duration)

	// テスト用のディレクトリを指定。
	rootDir := filepath.Join(os.Getenv("SERVER_ROOT"), "./data/test/ctg-assessment")
	command := "CTGRiskAssessmentor"
	parametersFile := "parameter.txt"

	s := &AssessmentService{nil, lib.GetDB(lib.ReadDBKey)}

	diagnosis, err := s.Execute(entity, diagnosisTime, duration, rootDir, command, parametersFile, "alg-0002", "1.0.0")

	assert.NoError(t, err)

	assert.EqualValues(t, 3, diagnosis.MeasurementId)
	assert.EqualValues(t, 11, *diagnosis.BaselineBpm)
	assert.EqualValues(t, 262, *diagnosis.MaximumRisk)
	assert.EqualValues(t, 6, len(diagnosis.Contents))

	parameters := map[string]interface{}{}

	assert.NoError(t, json.Unmarshal(diagnosis.Contents[0].Parameters, &parameters))
	assert.EqualValues(t, map[string]interface{}{
		"Acceleration": nil,
		"BpmArg": float64(150),
	}, parameters)
}