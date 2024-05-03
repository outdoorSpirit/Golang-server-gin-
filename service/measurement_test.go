package service

import (
	"fmt"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestServiceMeasurement_CollectForAssessment(t *testing.T) {
	db := lib.GetDB(lib.WriteDBKey)

	F.Truncate(db, "diagnosis_algorithm", "diagnosis", "measurement", "measurement_terminal", "patient", "hospital")

	hospitals := F.Insert(db, model.Hospital{}, 0, 2, func(i int, r F.Record) {
	}).([]*model.Hospital)

	patients := F.Insert(db, model.Patient{}, 0, 6, func(i int, r F.Record) {
		r["HospitalId"] = F.If(i <= 3, hospitals[0].Id, hospitals[1].Id)
	}).([]*model.Patient)

	terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 6, func(i int, r F.Record) {
		r["HospitalId"] = F.If(i <= 3, hospitals[0].Id, hospitals[1].Id)
	}).([]*model.MeasurementTerminal)

	diagnosisTime := time.Date(2021, time.January, 2, 3, 4, 5, 0, time.UTC)
	duration := time.Duration(10)*time.Minute
	interval := time.Duration(3)*time.Minute

	// 診断対象となる基準日時。20分前~5分前
	lastTime := diagnosisTime.Add(-time.Duration(5)*time.Minute)
	firstTime := diagnosisTime.Add(-time.Duration(20)*time.Minute)

	gen := func(hid int, r F.Record) {
		r["PatientId"] = patients[hid-1].Id
		r["TerminalId"] = terminals[hid-1].Id
		r["FirstTime"] = firstTime
		r["LastTime"] = lastTime
	}
	
	// m
	var m *model.Measurement
	ms := []interface{}{}
	mt := model.Measurement{}

	// 対象
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
	}).(*model.Measurement)
	ms = append(ms, m)

	// データの長さが足りない
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
		r["FirstTime"] = diagnosisTime.Add(-time.Duration(14)*time.Minute)
	}).(*model.Measurement)
	ms = append(ms, m)

	// first_timeがnull
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
		r["FirstTime"] = nil
	}).(*model.Measurement)
	ms = append(ms, m)

	// first_timeが対象期間より後
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
		r["FirstTime"] = diagnosisTime.Add(time.Duration(1)*time.Minute)
		r["LastTime"] = diagnosisTime.Add(time.Duration(15)*time.Minute)
	}).(*model.Measurement)
	ms = append(ms, m)

	// last_timeが対象期間より前
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
		r["LastTime"] = diagnosisTime.Add(-time.Duration(11)*time.Minute)
	}).(*model.Measurement)
	ms = append(ms, m)

	// 別病院の対象
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(2, r)
	}).(*model.Measurement)
	ms = append(ms, m)

	// 診断持ちの対象
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
	}).(*model.Measurement)
	ms = append(ms, m)
	correct := m

	// 診断持ち別病院の対象
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(3, r)
	}).(*model.Measurement)
	ms = append(ms, m)
	otherHospital := m

	// 最新の診断から、診断間隔分の時間が経過していない。
	// 本来last_timeとの比較により生じえないデータ。
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
	}).(*model.Measurement)
	ms = append(ms, m)
	shortInterval := m

	// 最新の診断より、last_timeが古い。
	m = F.Fixture(mt, len(ms), func(i int, r F.Record) {
		gen(1, r)
	}).(*model.Measurement)
	ms = append(ms, m)
	recentDiagnosis := m

	if e := db.Insert(ms...); e != nil {
		assert.FailNow(t, e.Error())
	}

	// 対象となる計測の持つ診断。range_untilmがinterval以前かつ計測のlast_time以前。
	// begin_timeから2分ごとに作成するため、range_untilは18,...,6となり、7データまでは問題ない。
	gend := func(m *model.Measurement, i int, r F.Record) {
		r["MeasurementId"] = m.Id
		r["RangeFrom"] = firstTime.Add(time.Duration((i-1)*2)*time.Minute)
		r["RangeUntil"] = firstTime.Add(time.Duration(i*2)*time.Minute)
	}

	F.Insert(db, model.Diagnosis{}, 0, 3, func(i int, r F.Record) {
		gend(correct, i, r)
	})

	F.Insert(db, model.Diagnosis{}, 0, 4, func(i int, r F.Record) {
		gend(otherHospital, i, r)
	})

	F.Insert(db, model.Diagnosis{}, 0, 3, func(i int, r F.Record) {
		gend(shortInterval, i, r)
		if i == 3 {
			r["RangeUntil"] = diagnosisTime.Add(-time.Duration(2)*time.Minute)
		}
	})

	F.Insert(db, model.Diagnosis{}, 0, 4, func(i int, r F.Record) {
		gend(recentDiagnosis, i, r)
		if i == 4 {
			r["RangeUntil"] = lastTime.Add(time.Duration(1)*time.Minute)
		}
	})

	// influxDBにデータ投入。
	influx := lib.GetInfluxDB()

	assert.NoError(t, influx.Delete("spiker", time.Unix(0, 0), time.Now(), ""))

	// 全ての計測のfirst~last間に30秒ごとにデータ追加。
	// 患者コードとマシンコードは利用しないため空文字にしておく。
	points := []lib.Point{}

	for _, m := range ms {
		mm := m.(*model.Measurement)
		if mm.FirstTime != nil && mm.LastTime != nil {
			ts := *mm.FirstTime
			index := 0
			for ts.Before(*mm.LastTime) {
				points = append(points,
					&model.HeartRate{mm.Id, "", "", mm.Id*100+index, ts},
					&model.TOCO{mm.Id, "", "", mm.Id*100+50+index, ts},
				)
				ts = ts.Add(time.Duration(30)*time.Second)
				index++
			}
		}
	}

	if es := influx.Insert("spiker", points...); len(es) > 0 {
		for _, e := range es {
			assert.NoError(t, e)
		}
		assert.FailNow(t, "Failed to insert influxDB records")
	}

	// サービス実行
	s := &MeasurementService{nil, lib.GetDB(lib.ReadDBKey), influx}

	entities, e := s.CollectForAssessment(diagnosisTime, duration, interval)
	assert.NoError(t, e)

	assert.EqualValues(t, 4, len(entities))

	assert.EqualValues(t, 1, entities[0].Id)
	assert.Nil(t, entities[0].LatestDiagnosis)
	// 心拍、TOCOともに10分前(duration)～5分前(last_time)の5分間の分。
	assert.EqualValues(t, 10, len(entities[0].HeartRates))
	for i, v := range entities[0].HeartRates {
		assert.EqualValues(t, diagnosisTime.Add(-duration+time.Duration(i*30)*time.Second), v.Timestamp)
		assert.EqualValues(t, 120+i, v.Value)
	}
	assert.EqualValues(t, 10, len(entities[0].TOCOs))
	for i, v := range entities[0].TOCOs {
		assert.EqualValues(t, diagnosisTime.Add(-duration+time.Duration(i*30)*time.Second), v.Timestamp)
		assert.EqualValues(t, 170+i, v.Value)
	}

	assert.EqualValues(t, 6, entities[1].Id)
	assert.Nil(t, entities[1].LatestDiagnosis)
	assert.EqualValues(t, 10, len(entities[1].HeartRates))
	assert.EqualValues(t, 10, len(entities[1].TOCOs))

	assert.EqualValues(t, 7, entities[2].Id)
	assert.EqualValues(t, 3, entities[2].LatestDiagnosis.Id)
	assert.EqualValues(t, 10, len(entities[2].HeartRates))
	assert.EqualValues(t, 10, len(entities[2].TOCOs))

	assert.EqualValues(t, 8, entities[3].Id)
	assert.EqualValues(t, 7, entities[3].LatestDiagnosis.Id)
	assert.EqualValues(t, 10, len(entities[3].HeartRates))
	assert.EqualValues(t, 10, len(entities[3].TOCOs))
}

func TestServiceMeasurement_EnsureNotExists(t *testing.T) {
	db := lib.GetDB(lib.WriteDBKey)

	F.Truncate(db, "measurement", "measurement_terminal", "patient", "hospital")

	hospitals := F.Insert(db, model.Hospital{}, 0, 2, func(i int, r F.Record) {
	}).([]*model.Hospital)

	patients := F.Insert(db, model.Patient{}, 0, 6, func(i int, r F.Record) {
		r["HospitalId"] = F.If(i <= 3, hospitals[0].Id, hospitals[1].Id)
	}).([]*model.Patient)

	terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 6, func(i int, r F.Record) {
		r["HospitalId"] = F.If(i <= 3, hospitals[0].Id, hospitals[1].Id)
	}).([]*model.MeasurementTerminal)

	// 診断対象となる基準日時。20分前~5分前
	now := time.Now()
	lastTime := now.Add(-time.Duration(5)*time.Minute)
	firstTime := now.Add(-time.Duration(20)*time.Minute)

	// h - m     - (p, t)
	// 1 - 1(c1) - (1, 1)
	//   - 2(c2) - (2, 2)
	//   - 3(c3) - (3, 3)
	// 2 - 3(c4) - (4, 4)
	ms := F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
		r["Code"] = fmt.Sprintf("c%d", i)
		r["PatientId"] = patients[i-1].Id
		r["TerminalId"] = terminals[i-1].Id
		r["FirstTime"] = firstTime
		r["LastTime"] = lastTime
	}).([]*model.Measurement)

	// influxDBにデータ投入。
	influx := lib.GetInfluxDB()

	assert.NoError(t, influx.DeleteAll("spiker", ""))

	// 全ての計測のfirst~last間に30秒ごとにデータ追加。
	// 患者コードとマシンコードは利用しないため空文字にしておく。
	points := []lib.Point{}

	for _, mm := range ms {
		if mm.FirstTime != nil && mm.LastTime != nil {
			ts := *mm.FirstTime
			index := 0
			for ts.Before(*mm.LastTime) {
				points = append(points,
					&model.HeartRate{mm.Id, "", "", mm.Id*100+index, ts},
					&model.TOCO{mm.Id, "", "", mm.Id*100+50+index, ts},
				)
				ts = ts.Add(time.Duration(30)*time.Second)
				index++
			}
		}
	}

	assert.EqualValues(t, 240, len(points))

	if es := influx.Insert("spiker", points...); len(es) > 0 {
		for _, e := range es {
			assert.NoError(t, e)
		}
		assert.FailNow(t, "Failed to insert influxDB records")
	}

	// サービス実行。
	tx, err := db.Begin()

	assert.NoError(t, err)

	service := &MeasurementTxService{nil, tx, influx}

	assert.NoError(t, service.EnsureNotExists(1, "c2", true))

	// DBからの削除を確認。
	noRows := tx.SelectOne(&model.Measurement{}, `SELECT * FROM measurement WHERE id = 2`)

	assert.EqualValues(t, sql.ErrNoRows, noRows)

	// influxDBからの削除を確認。
	actuals := []lib.Point{}

	influx.Select(
		`from(bucket:"spiker") |> range(start:-1h) |> group()`,
		lib.PointConsumer(func(p lib.Point, field string) error {
			actuals = append(actuals, p)
			return nil
		}),
	)

	assert.EqualValues(t, 180, len(actuals))

	for _, p := range actuals {
		switch pp := p.(type) {
		case *model.HeartRate:
			assert.True(t, pp.MeasurementId == 1 || pp.MeasurementId == 3 || pp.MeasurementId == 4)
		case *model.TOCO:
			assert.True(t, pp.MeasurementId == 1 || pp.MeasurementId == 3 || pp.MeasurementId == 4)
		default:
			assert.FailNow(t, "Unexpected point: %v", pp)
		}
	}
}