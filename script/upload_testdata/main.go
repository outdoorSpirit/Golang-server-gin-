package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"gopkg.in/gorp.v2"

	"github.com/spiker/spiker-server/config"
	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
)

const (
	hospitalUuid = "aab5e439-2947-4def-9df7-e8fbe1cad07f"

	doctorLoginId = "testdata-doctor"
	doctorPassword = "testdata"

	machineIdFormat = "test#m-%04d"
	patientIdFormat = "%s-patient#%04d"

	algorithmName = "testdata-algorithm"
)

var (
	algorithm *model.DiagnosisAlgorithm = nil
)

func prepareAlgorithm(tx *gorp.Transaction) error {
	alg := model.DiagnosisAlgorithm{}

	if e := tx.SelectOne(&alg, `SELECT * FROM diagnosis_algorithm WHERE name = $1`, algorithmName); e != nil {
		if e == sql.ErrNoRows {
			algorithm = &model.DiagnosisAlgorithm{
				Name: algorithmName,
				Version: "0.0.1",
				Memo: "",
				CreatedAt: time.Now(),
				ModifiedAt: time.Now(),
			}

			if e := tx.Insert(algorithm); e != nil {
				return e
			}
		} else {
			return e
		}
	} else {
		algorithm = &alg
	}

	return nil
}

func prepareHospital(tx *gorp.Transaction) (*model.Hospital, error) {
	hospital := model.Hospital{}

	if e := tx.SelectOne(&hospital, `SELECT * FROM hospital WHERE uuid = $1`, hospitalUuid); e != nil {
		if e == sql.ErrNoRows {
			hospital = model.Hospital{
				Uuid: hospitalUuid,
				Topic: "test",
				Name: "Test data hospital",
				Memo: "",
				CreatedAt: time.Now(),
				ModifiedAt: time.Now(),
			}

			if e := tx.Insert(&hospital); e != nil {
				return nil, e
			}

			ds := S.DoctorTxService{nil, tx}

			if _, e := ds.Create(hospital.Id, doctorLoginId, doctorPassword, "Doctor for testdata"); e != nil {
				return nil, e
			}

			return &hospital, nil
		} else {
			return nil, e
		}
	}

	return &hospital, nil
}

func cleanPreviousData(hospital *model.Hospital, tx *gorp.Transaction, influx lib.InfluxDBClient) error {
	// 既存の計測を全て取得。
	measurements := []*model.Measurement{}

	if _, e := tx.Select(
		&measurements,
		`SELECT m.* FROM measurement AS m INNER JOIN patient AS p ON m.patient_id = p.id WHERE p.hospital_id = $1`,
		hospital.Id,
	); e != nil {
		return e
	}

	// 端末は10個までという前提でInfluxDBからレコードを削除。
	for i := 1; i <= 10; i++ {
		if e := influx.Delete(
			"spiker",
			time.Unix(0, 0),
			time.Now().Add(time.Duration(24*365*10)*time.Hour),
			fmt.Sprintf(`machine="%s"`, fmt.Sprintf(machineIdFormat, i)),
		); e != nil {
			return e
		}
	}

	// 計測を削除。
	holder := []interface{}{}

	for _, m := range measurements {
		holder = append(holder, m)
	}

	if len(holder) > 0 {
		if _, e := tx.Delete(holder...); e != nil {
			return e
		}
	}

	return nil
}

type CTGEntry struct {
	mid       string
	pid       string
	timestamp int64
	value     int
}

func (e CTGEntry) Record() []string {
	return []string{e.mid, e.pid, fmt.Sprintf("%d", e.value), fmt.Sprintf("%d", e.timestamp)}
}

func generateCTGData(mt C.MeasurementType, current time.Time, mc int, ppm int, duration time.Duration) []CTGEntry {
	entries := []CTGEntry{}

	genHR := func() []int {
		len := 40 + rand.Intn(30*4)
		vol := float64(5 + rand.Intn(25))

		series := []int{}
		for i := 0; i < len; i++ {
			v := int(vol * math.Sin(math.Pi * float64(i) / float64(len)))
			series = append(series, v)
		}

		return series
	}

	genUC := func() []int {
		len := 20 + rand.Intn(10*4)

		series := make([]int, len, len)
		if rand.Intn(10) <= 5 {
			return series
		}

		vol := float64(5 + rand.Intn(20))

		u := float64(len) / 2.0
		sig := float64(len) / 6.0
		den := 2.0*sig*sig
		for i := 0; i < len; i++ {
			x := float64(i)
			v := int(math.Abs(vol * math.Exp(-(x-u)*(x-u)/den)))
			series[i] = v
		}

		return series
	}

	var gen func() []int
	var base int

	if mt == C.MeasurementTypeHeartRate {
		gen = genHR
		base = 150
	} else {
		gen = genUC
		base = 0
	}

	for i := 1; i <= mc; i++ {
		mid := fmt.Sprintf(machineIdFormat, i)

		begin := current.Add(time.Duration(-i*15)*time.Minute).Add(-time.Hour)

		for j := 1; j <= ppm; j++ {
			pid := fmt.Sprintf(patientIdFormat, mid, j)

			// 各計測1時間分のデータ。毎秒4つ(250msごと)。
			start := begin.Add(time.Duration(-j*20)*time.Minute)

			var series1 []int = nil
			var series2 []int = nil

			var k int64 = 0
			dataNum := 4 * int64(duration.Seconds())

			for k < dataNum {
				ts := start.Add(time.Duration(250*k)*time.Millisecond)

				if series1 == nil || len(series1) == 0 {
					series1 = gen()
				}
				if series2 == nil || len(series2) == 0 {
					series2 = gen()
				}

				entries = append(entries, CTGEntry{
					mid, pid, ts.UnixNano()/1000000, base + series1[0] + series2[0],
				})

				series1 = series1[1:]
				series2 = series2[1:]

				k++
			}
		}
	}

	return entries
}

func uploadCsv(
	hospital *model.Hospital,
	tx *gorp.Transaction,
	influx lib.InfluxDBClient,
	current time.Time,
	mc int,
	ppm int,
	duration time.Duration,
) ([]CTGEntry, error) {
	hrEntries := generateCTGData(C.MeasurementTypeHeartRate, current, mc, ppm, duration)
	ucEntries := generateCTGData(C.MeasurementTypeTOCO, current, mc, ppm, duration)

	data := []S.CTGData{}

	for i, hr := range hrEntries {
		if i >= len(ucEntries) {
			break
		}

		uc := ucEntries[i]

		data = append(data, S.CTGData{
			MachineId: hr.mid,
			PatientId: hr.pid,
			FHR1: fmt.Sprintf("%d", hr.value),
			UC: fmt.Sprintf("%d", uc.value),
			FHR2: "0",
			Timestamp: fmt.Sprintf("%d", hr.timestamp),
		})
	}

	service := &S.DataTxService{nil, tx, influx}

	if stats, e := service.RegisterCTGData(hospital.Id, data); e != nil {
		return nil, e
	} else {
		fmt.Printf("FHR1 success: %d\n", stats.SuccessFHR1)
		fmt.Printf("FHR1 failed: %d\n", len(stats.FailureFHR1))
		fmt.Printf("UC success: %d\n", stats.SuccessUC)
		fmt.Printf("UC failed: %d\n", len(stats.FailureUC))
		return hrEntries, nil
	}
}

func generateEvent() (map[string]interface{}, *int, *int) {
	params := map[string]interface{}{}
	var baseline *int = nil
	var risk *int = nil

	switch rand.Intn(3) {
	case 0:
		et := C.BaselineEvents[rand.Intn(len(C.BaselineEvents))]
		vt := C.BaselineVariabilityEvents[rand.Intn(len(C.BaselineVariabilityEvents))]
		bl := 120+rand.Intn(50)
		baseline = &bl
		params[string(et)] = bl
		params[string(vt)] = rand.Intn(30)
		if rand.Intn(10) >= 5 {
			r := rand.Intn(5)+1
			risk = &r
			params["Risk"] = r
		}
	case 1:
		dt := C.DecelerationEvents[rand.Intn(len(C.DecelerationEvents))]
		params[string(dt)] = nil
		if rand.Intn(10) >= 2 {
			r := rand.Intn(5)+1
			risk = &r
			params["Risk"] = r
		}
	case 2:
		params[string(C.CTG_Acceleration)] = nil
	}

	return params, baseline, risk
}

func generateDiagnoses(entries []CTGEntry, tx *gorp.Transaction) error {
	patients := map[string]*struct{
		from int64
		until int64
	}{}

	for _, e := range entries {
		if _, be := patients[e.pid]; !be {
			patients[e.pid] = &struct{ from int64; until int64 }{e.timestamp, e.timestamp}
		} else {
			patients[e.pid].until = e.timestamp
		}
	}

	// 計測範囲にランダムに診断を生成。
	entities := []*model.DiagnosisEntity{}

	for pid, s := range patients {
		measurement := model.Measurement{}

		if e := tx.SelectOne(&measurement, `SELECT * FROM measurement WHERE code = $1`, pid); e != nil {
			return e
		}

		diagnosis := &model.DiagnosisEntity{
			Diagnosis: &model.Diagnosis{
				MeasurementId: measurement.Id,
				BaselineBpm: nil,
				MaximumRisk: nil,
				Memo: "",
				RangeFrom: time.Unix(int64(s.from/1000), (s.from%1000)*1000000),
				RangeUntil: time.Unix(int64(s.until/1000), (s.until%1000)*1000000),
				CreatedAt: time.Now(),
				ModifiedAt: time.Now(),
			},
			Algorithm: algorithm,
			Contents: []*model.DiagnosisContent{},
		}

		// 3~8分に1つのイベント。
		interval := time.Duration(rand.Intn(6)+3)*time.Minute
		eventFrom := s.from + interval.Nanoseconds()/1000000

		var baselineBpm *int = nil
		var maximumRisk *int = nil

		for eventFrom < s.until {
			// 長さは30秒~2分。
			eventUntil := eventFrom + int64(rand.Intn(90)+30)*1000

			// 基線、徐脈、頻脈はランダム。リスクの有無もランダム。
			params, baseline, risk := generateEvent()

			if baseline != nil && (baselineBpm == nil || *baseline > *baselineBpm) {
				baselineBpm = baseline
			}
			if risk != nil && (maximumRisk == nil || *risk > *maximumRisk) {
				maximumRisk = risk
			}

			dc := &model.DiagnosisContent{
				DiagnosisId: 0,
				Risk: risk,
				RangeFrom: time.Unix(int64(eventFrom/1000), (eventFrom%1000)*1000000),
				RangeUntil: time.Unix(int64(eventUntil/1000), (eventUntil%1000)*1000000),
				Parameters: nil,
			}

			if j, e := json.Marshal(params); e != nil {
				return e
			} else {
				dc.Parameters = model.JSON(j)
			}

			diagnosis.Contents = append(diagnosis.Contents, dc)

			interval = time.Duration(rand.Intn(10)+5)*time.Minute
			eventFrom += interval.Nanoseconds()/1000000
		}

		diagnosis.BaselineBpm = baselineBpm
		diagnosis.MaximumRisk = maximumRisk

		entities = append(entities, diagnosis)
	}

	if len(entities) > 0 {
		service := &S.DiagnosisTxService{nil, tx}

		if e := service.Register(entities); e != nil {
			return e
		}
	}

	return nil
}

func execute(tx *gorp.Transaction, influx lib.InfluxDBClient, current time.Time, mc int, ppm int, duration time.Duration) error {
	if e := prepareAlgorithm(tx); e != nil {
		return e
	}

	hospital, err := prepareHospital(tx)

	if err != nil {
		return err
	}

	if e := cleanPreviousData(hospital, tx, influx); e != nil {
		return e
	}

	entries, err := uploadCsv(hospital, tx, influx, current, mc, ppm, duration)

	if err != nil {
		return err
	}

	// 診断
	if e := generateDiagnoses(entries, tx); e != nil {
		return e
	}

	return nil
}

func main() {
	config.SetupAll()

	db := lib.GetDB(lib.WriteDBKey)

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to open transaction: %v\n", err)
	}

	influx := lib.GetInfluxDB()

	if e := execute(tx, influx, time.Now(), 2, 3, time.Duration(120)*time.Minute); e != nil {
		log.Println(e)
		tx.Rollback()
	} else {
		tx.Commit()
	}
}