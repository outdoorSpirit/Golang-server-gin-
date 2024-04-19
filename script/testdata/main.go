package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"gopkg.in/gorp.v2"

	"github.com/spiker/spiker-server/config"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
)

// デフォルトでは、
// 2021-10-01 12:00:00 UTC
// を基準として、各患者に
// 06:00:00-
// 06:00:00-
// 09:00:00-
// 09:30:00-
// 11:00:00-
// の1時間分のデータが投入される。

var (
	hospitalUUIDs = []string{
		"7070b4fd-18c1-4bed-aefe-85e8f31b1ab9",
		"828e0779-2f57-483f-8004-c2ff0ffb04ee",
		"ecb683b0-52a5-43ae-96e1-ab06e29af515",
	}

	hospitalTopics = []string{
		"afb4c6af-aa68-4573-8728-19f8d17c5f69",
		"34395acc-55f8-4d26-9762-2e57ef84299f",
		"fca27086-8cdf-4d07-b138-f5f3115a9d78",
	}

	doctorTopics = []string{
		"a38714ca-df40-4596-a6b2-7f965287ad94",
		"1e6bb023-ee1a-4dc1-a939-6c1c36751cfd",
		"24baf5c9-7b44-40b2-bd04-6d4ef03d6e33",
		"5f09eba1-8722-4c78-bd25-9181a2658545",
		"62c1bd05-8cd0-48b5-88d7-6746de3ac915",
		"3d2b4b57-71a3-4da8-b9f2-f716f484e576",
	}

	ctgApiKeys = []string{
		"14aeec59-ab00-456a-83ed-bad1be2f8de3",
		"230d4a21-339e-4237-83ef-bad2e61f367a",
		"0b606c44-4e98-4953-b2b1-a7469cfd3e6c",
	}
)

func insertMasterData(db *gorp.Transaction, now time.Time) ([]*model.MeasurementEntity, error) {
	var records []interface{}

	// 病院。
	hospitals := []*model.Hospital{}
	records = []interface{}{}

	for i := 0; i < 3; i++ {
		m := &model.Hospital{
			Uuid:       hospitalUUIDs[i],
			Topic:      hospitalTopics[i],
			Name:       fmt.Sprintf("Hospital No.%d", i+1),
			Memo:       "",
			CreatedAt:  now,
			ModifiedAt: now,
		}

		hospitals = append(hospitals, m)
		records = append(records, m)
	}

	log.Printf("Insert %d hospitals...\n", len(records))
	if e := db.Insert(records...); e != nil {
		return nil, e
	}

	// CTG-API認証。
	records = []interface{}{}

	for i := 0; i < 3; i++ {
		m := &model.CTGAuthentication{
			HospitalId: hospitals[i].Id,
			ApiKey:     ctgApiKeys[i],
			CreatedAt:  now,
		}

		records = append(records, m)
	}

	log.Printf("Insert %d CTG API authentications...\n", len(records))
	if e := db.Insert(records...); e != nil {
		return nil, e
	}

	// 計測アルゴリズム。
	log.Println("Insert a diagnosis algorithm ...")
	if e := db.Insert(&model.DiagnosisAlgorithm{
		Name:       "CTGRiskAssessmentor",
		Version:    "1.1.0",
		Memo:       "",
		CreatedAt:  now,
		ModifiedAt: now,
	}); e != nil {
		return nil, e
	}

	// 医師。
	records = []interface{}{}

	for i := 0; i < 6; i++ {
		hi := int(i / 3)

		m := &model.Doctor{
			HospitalId:   hospitals[hi].Id,
			LoginId:      fmt.Sprintf("doctor%04d", i+1),
			Password:     fmt.Sprintf("pass%04d", i+1),
			TokenVersion: fmt.Sprintf("token-%04d", i+1),
			Topic:        doctorTopics[i],
			Name:         fmt.Sprintf("Doctor No.%d", i+1),
			CreatedAt:    now,
			ModifiedAt:   now,
		}

		records = append(records, m)
	}

	log.Printf("Insert %d doctors...\n", len(records))
	if e := db.Insert(records...); e != nil {
		return nil, e
	}

	log.Println("Encrypt doctor passwords...")
	if _, e := db.Exec(`UPDATE doctor SET password = encode(digest(password, 'sha256'), 'hex')`); e != nil {
		return nil, e
	}

	// 患者。
	patients := []*model.Patient{}
	records = []interface{}{}

	for i := 0; i < 10; i++ {
		hi := int(i / 5)

		m := &model.Patient{
			HospitalId: hospitals[hi].Id,
			CreatedAt:  now,
			ModifiedAt: now,
		}

		patients = append(patients, m)
		records = append(records, m)
	}

	log.Printf("Insert %d patients...\n", len(records))
	if e := db.Insert(records...); e != nil {
		return nil, e
	}

	// 計測端末。
	terminals := []*model.MeasurementTerminal{}
	records = []interface{}{}

	for i := 0; i < 10; i++ {
		hi := int(i / 5)

		m := &model.MeasurementTerminal{
			HospitalId: hospitals[hi].Id,
			Code:       fmt.Sprintf("EDAN#%04d", i+1),
			Memo:       "",
			CreatedAt:  now,
			ModifiedAt: now,
		}

		terminals = append(terminals, m)
		records = append(records, m)
	}

	log.Printf("Insert %d terminals...\n", len(records))
	if e := db.Insert(records...); e != nil {
		return nil, e
	}

	// 計測記録。現状は患者と一致させる。
	entities := []*model.MeasurementEntity{}
	records = []interface{}{}

	tis := map[int]int{
		0: 0, 1: 1, 2: 2, 3: 0, 4: 0,
		5: 5, 6: 6, 7: 7, 8: 5, 9: 5,
	}
	minutes := map[int]int{
		0: 300, 1: 300, 2: 180, 3: 150, 4: 60,
		5: 300, 6: 300, 7: 180, 8: 150, 9: 60,
	}
	for i := 0; i < 10; i++ {
		first := now.Add(-time.Duration(minutes[i])*time.Minute)
		last := first.Add(time.Duration(1)*time.Hour)

		m := &model.Measurement{
			Code:       fmt.Sprintf("%s-%s-%s", terminals[tis[i]].Code, i+1, first.Format("20060102150405")),
			PatientId:  patients[i].Id,
			TerminalId: terminals[tis[i]].Id,
			FirstTime:  &first,
			LastTime:   &last,
			CreatedAt:  now,
			ModifiedAt: now,
		}

		entities = append(entities, &model.MeasurementEntity{
			Measurement: m,
			Terminal:    terminals[tis[i]],
			Patient:     patients[i],
		})
		records = append(records, m)
	}

	log.Printf("Insert %d measurements...\n", len(records))
	if e := db.Insert(records...); e != nil {
		return nil, e
	}

	return entities, nil
}

func insertCTGData(influx lib.InfluxDBClient, measurements []*model.MeasurementEntity) error {
	// 各計測のfirst~last間に、4Hzのデータを与える。
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
			v := int(vol * math.Exp(-(x-u)*(x-u)/den))
			series[i] = v
		}

		return series
	}

	for _, m := range measurements {
		// 心拍
		// 99.9%:10~40秒程度の変化(振幅5~30), 0.1%:80程度までの急降下
		t := *m.FirstTime

		ser1 := genHR()
		ser2 := genHR()

		points := []lib.Point{}

		for !t.After(*m.LastTime) {
			if len(ser1) == 0 {
				ser1 = genHR()
			}
			if len(ser2) == 0 {
				ser2 = genHR()
			}

			points = append(points, &model.HeartRate{
				MeasurementId: m.Id,
				PatientCode:   m.Code,
				MachineCode:   m.Terminal.Code,
				Value:         150+ser1[0]+ser2[0],
				Timestamp:     t,
			})

			ser1 = ser1[1:]
			ser2 = ser2[1:]

			t = t.Add(time.Duration(250)*time.Millisecond)
		}

		// TOCO
		// 50%:0持続, 50%:5~10秒程度の変化(振幅10~20)
		t = *m.FirstTime

		ser1 = genUC()
		ser2 = genUC()

		for !t.After(*m.LastTime) {
			if len(ser1) == 0 {
				ser1 = genUC()
			}
			if len(ser2) == 0 {
				ser2 = genUC()
			}

			points = append(points, &model.TOCO{
				MeasurementId: m.Id,
				PatientCode:   m.Code,
				MachineCode:   m.Terminal.Code,
				Value:         ser1[0]+ser2[0],
				Timestamp:     t,
			})

			ser1 = ser1[1:]
			ser2 = ser2[1:]

			t = t.Add(time.Duration(250)*time.Millisecond)
		}

		if pn := len(points); pn > 1000000 {
			return fmt.Errorf("Too many points: %d", pn)
		} else {
			log.Printf("Insert %d points for measurement #%d\n", pn, m.Id)

			if es := influx.Insert("spiker", points...); len(es) != 0 {
				return fmt.Errorf("%d errors happened in insertion, first error: %v", len(es), es[0])
			}
		}
	}

	return nil
}

func main() {
	// 設定
	os.Setenv("SERVER_ENV", "dev")

	config.SetupAll()

	// コマンドライン引数
	var noTruncate bool
	flag.BoolVar(&noTruncate, "nt", false, "Flag not to clear existing data")

	var currentTime string
	flag.StringVar(&currentTime, "now", "", "Current UTC time of master data in YYYYMMDDhhmmss format")

	flag.Parse()

	var now time.Time

	if currentTime == "" {
		now = time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC).UTC()
	} else if t, e := time.Parse("20060102150405", currentTime); e != nil {
		log.Fatal("Failed to parse current time: %v\n", e)
	} else {
		now = t.UTC()
	}

	// 処理実行
	db, err := lib.GetDB(lib.WriteDBKey).Begin()
	if err != nil {
		log.Fatalf("Failed to open transaction: %v\n", err)
	}

	influx := lib.GetInfluxDB()

	// データ削除。
	log.Println("---------- Truncate ----------")

	if noTruncate {
		log.Println("Skipped by CLI option.")
	} else {
		for _, t := range []string{
			"diagnosis_algorithm",
			"diagnosis",
			"measurement",
			"measurement_terminal",
			"patient",
			"doctor",
			"ctg_authentication",
			"hospital",
		} {
			log.Printf("Truncate %s...\n", t)
			if _, e := db.Exec(fmt.Sprintf(`TRUNCATE %s RESTART IDENTITY CASCADE`, t)); e != nil {
				db.Rollback()
				log.Fatalf("Failed to truncate table %s: %v\n", t, e)
			}
		}

		beginTime := time.Unix(0, 0)
		endTime := time.Now()
		if now.After(endTime) {
			endTime = now
		}

		log.Println("Truncate data in InfluxDB [%v ~ %v]...", beginTime, endTime)

		if e := influx.Delete("spiker", beginTime, endTime, ""); e != nil {
			db.Rollback()
			log.Fatalf("Failed to truncate influxDB data: %v\n", e)
		}
	}

	// マスタデータ投入。
	log.Println("---------- Insert records to DB ----------")

	var measurements []*model.MeasurementEntity

	if ms, e := insertMasterData(db, now); e != nil {
		db.Rollback()

		log.Fatalf("Failed to insert: %v\n", e)
	} else {
		measurements = ms
	}

	// InfluxDB投入。
	log.Println("---------- Insert records to InfluxDB ----------")

	if e := insertCTGData(influx, measurements); e != nil {
		db.Rollback()

		log.Fatalf("Failed to insert: %v\n", e)
	}

	db.Commit()

	log.Println("Success")
}