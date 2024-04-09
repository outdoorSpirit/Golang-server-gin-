package ctg

import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	//"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestCTG_Upload(t *testing.T) {
	auth := (&F.CTGFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)
	influx := lib.GetInfluxDB()

	truncate := func(req *http.Request) {
		F.Truncate(db, "measurement", "patient", "measurement_terminal")
		influx.Delete("spiker", time.Unix(0, 0), time.Now(), "")
	}

	million := int64(1000000)

	beginTime := time.Now().Add(time.Duration(-1)*time.Hour)
	nano := int64(beginTime.UnixNano() / million) * million
	beginTime = time.Unix(0, nano).UTC()

	timestamp := func(sec int) string {
		return fmt.Sprintf("%d", int64(beginTime.Add(time.Duration(sec)*time.Second).UnixNano() / million))
	}

	httpTests := test.HttpTests{
		{
			Name:    "空アップロード",
			Method:  http.MethodPost,
			Path:    "/ctg/data",
			Token:   auth.Token(3),
			Body:    test.JsonBody([]interface{}{}),
			Prepare: test.Prepares(truncate),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &uploadResponse{}).(*uploadResponse)

				assert.EqualValues(t, 0, res.Success)
				assert.EqualValues(t, 0, res.Failure)
			},
		},
		{
			Name:    "アップロード",
			Method:  http.MethodPost,
			Path:    "/ctg/data",
			Token:   auth.Token(3),
			Body:    test.JsonBody([]map[string]interface{}{
				// 新規計測。端末も新規。先頭と末尾のタイムスタンプが記録される。
				map[string]interface{}{
					"Patient ID": "m0", "Machine ID": "t0", "Timestamp": timestamp(1),
					"FHR1": "10", "UC": "100", "FHR2": "1000",
				},
				map[string]interface{}{
					"Patient ID": "m0", "Machine ID": "t0", "Timestamp": timestamp(3),
					"FHR1": "30", "UC": "300", "FHR2": "3000",
				},
				map[string]interface{}{
					"Patient ID": "m0", "Machine ID": "t0", "Timestamp": timestamp(2),
					"FHR1": "20", "UC": "200", "FHR2": "2000",
				},
				// 新規計測。端末は既存。
				map[string]interface{}{
					"Patient ID": "m5", "Machine ID": "t2", "Timestamp": timestamp(4),
					"FHR1": "40", "UC": "400", "FHR2": "4000",
				},
				map[string]interface{}{
					"Patient ID": "m5", "Machine ID": "t2", "Timestamp": timestamp(5),
					"FHR1": "50", "UC": "500", "FHR2": "5000",
				},
				// 既存の計測。末尾のタイムスタンプが更新される。
				map[string]interface{}{
					"Patient ID": "m2", "Machine ID": "t2", "Timestamp": timestamp(6),
					"FHR1": "60", "UC": "600", "FHR2": "6000",
				},
				map[string]interface{}{
					"Patient ID": "m2", "Machine ID": "t2", "Timestamp": timestamp(7),
					"FHR1": "70", "UC": "700", "FHR2": "7000",
				},
				// 新規計測だが同一コードの計測が別病院にある。
				map[string]interface{}{
					"Patient ID": "m1", "Machine ID": "t5", "Timestamp": timestamp(8),
					"FHR1": "80", "UC": "800", "FHR2": "8000",
				},
				// 新規計測。端末も新規だが同一コードの端末が別病院にある。
				map[string]interface{}{
					"Patient ID": "m6", "Machine ID": "t1", "Timestamp": timestamp(9),
					"FHR1": "90", "UC": "900", "FHR2": "9000",
				},
			}),
			Prepare: test.Prepares(truncate, func(req *http.Request) {
				// h - m  - t
				// 1 - m1 - t1
				// 2 - m2 - t2
				// 2 - m3 - t3
				// 3 - m4 - t4
				ts := []struct{ hid int; tc string }{
					{1, "t1"}, {2, "t2"}, {2, "t3"}, {3, "t4"},
				}

				terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 4, func(i int, r F.Record) {
					r["HospitalId"] = ts[i-1].hid
					r["Code"] = ts[i-1].tc
				}).([]*model.MeasurementTerminal)

				ms := []struct{ mc string; ti int }{
					{"m1", 0}, {"m2", 1}, {"m3", 2}, {"m4", 3},
				}

				patients := F.Insert(db, model.Patient{}, 0, 4, func(i int, r F.Record) {
					r["HospitalId"] = terminals[ms[i-1].ti].HospitalId
					r["Name"] = fmt.Sprintf("患者-%d", i)
				}).([]*model.Patient)

				F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
					r["Code"] = ms[i-1].mc
					r["PatientId"] = patients[i-1].Id
					r["TerminalId"] = terminals[ms[i-1].ti].Id
					r["FirstTime"] = beginTime
					r["LastTime"] = beginTime
				})
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &uploadResponse{}).(*uploadResponse)

				assert.EqualValues(t, 9, res.Success)
				assert.EqualValues(t, 0, res.Failure)

				// 計測端末t0,t5,t1がhospital=2に増える。
				assert.EqualValues(t, 7, F.Count(t, db, "measurement_terminal", nil))

				terminals := []*model.MeasurementTerminal{}
				F.Select(t, db, &terminals, F.NewQuery("").Asc("id"))
				assert.EqualValues(t, 7, len(terminals))

				expectTeminal := []struct{id int; code string; hid int}{
					{5, "t0", 2}, {6, "t5", 2}, {7, "t1", 2},
				}
				for i, exp := range expectTeminal {
					assert.EqualValues(t, exp.id, terminals[i+4].Id)
					assert.EqualValues(t, exp.code, terminals[i+4].Code)
					assert.EqualValues(t, exp.hid, terminals[i+4].HospitalId)
				}

				// 患者、計測が4つ増える。
				assert.EqualValues(t, 8, F.Count(t, db, "patient", nil))
				assert.EqualValues(t, 8, F.Count(t, db, "measurement", nil))

				patients := []*model.Patient{}
				measurements := []*model.Measurement{}

				F.Select(t, db, &patients, F.NewQuery("").Asc("id"))
				F.Select(t, db, &measurements, F.NewQuery("").Asc("id"))

				assert.EqualValues(t, 8, len(patients))
				assert.EqualValues(t, 8, len(measurements))

				expectPatient := []struct{id int; hid int}{
					{5, 2}, {6, 2}, {7, 2}, {8, 2},
				}
				for i, exp := range expectPatient {
					assert.EqualValues(t, exp.id, patients[i+4].Id)
					assert.EqualValues(t, exp.hid, patients[i+4].HospitalId)
				}

				expectMeasurement := []struct{id int; pid int; tid int; code string; b int; e int}{
					{5, 5, 5, "m0", 1, 3}, {6, 6, 2, "m5", 4, 5}, {7, 7, 6, "m1", 8, 8}, {8, 8, 7, "m6", 9, 9},
				}
				for i, exp := range expectMeasurement {
					assert.EqualValues(t, exp.id, measurements[i+4].Id)
					assert.EqualValues(t, exp.pid, measurements[i+4].PatientId)
					assert.EqualValues(t, exp.tid, measurements[i+4].TerminalId)
					assert.EqualValues(t, exp.code, measurements[i+4].Code)
					assert.EqualValues(t, beginTime.Add(time.Duration(exp.b)*time.Second).UnixNano(), measurements[i+4].FirstTime.UnixNano())
					assert.EqualValues(t, beginTime.Add(time.Duration(exp.e)*time.Second).UnixNano(), measurements[i+4].LastTime.UnixNano())
				}

				assert.EqualValues(t, beginTime.UnixNano(), measurements[1].FirstTime.UnixNano())
				assert.EqualValues(t, beginTime.Add(time.Duration(7)*time.Second).UnixNano(), measurements[1].LastTime.UnixNano())

				// InfluxDBに9レコード。
				hrs := []*model.HeartRate{}
				iq := `from(bucket:"spiker")
						|> range(start:-3h)
						|> filter(fn: (r) => r._measurement == "heartrate")
						|> group()
						|> sort(columns:["_time"])`
				influx.Select(iq, lib.PointConsumer(func(p lib.Point, f string) error {
					hr, ok := p.(*model.HeartRate)
					assert.True(t, ok)
					hrs = append(hrs, hr)
					return nil
				}))

				assert.EqualValues(t, 9, len(hrs))

				expectedHeartRates := []model.HeartRate{
					model.HeartRate{5, "m0", "t0", 10, beginTime.Add(time.Duration(1)*time.Second)},
					model.HeartRate{5, "m0", "t0", 20, beginTime.Add(time.Duration(2)*time.Second)},
					model.HeartRate{5, "m0", "t0", 30, beginTime.Add(time.Duration(3)*time.Second)},
					model.HeartRate{6, "m5", "t2", 40, beginTime.Add(time.Duration(4)*time.Second)},
					model.HeartRate{6, "m5", "t2", 50, beginTime.Add(time.Duration(5)*time.Second)},
					model.HeartRate{2, "m2", "t2", 60, beginTime.Add(time.Duration(6)*time.Second)},
					model.HeartRate{2, "m2", "t2", 70, beginTime.Add(time.Duration(7)*time.Second)},
					model.HeartRate{7, "m1", "t5", 80, beginTime.Add(time.Duration(8)*time.Second)},
					model.HeartRate{8, "m6", "t1", 90, beginTime.Add(time.Duration(9)*time.Second)},
				}

				for i, exp := range expectedHeartRates {
					assert.EqualValues(t, exp, *hrs[i])
				}

				tocos := []*model.TOCO{}
				iq = `from(bucket:"spiker")
						|> range(start:-3h)
						|> filter(fn: (r) => r._measurement == "toco")
						|> group()
						|> sort(columns:["_time"])`
				influx.Select(iq, lib.PointConsumer(func(p lib.Point, f string) error {
					tc, ok := p.(*model.TOCO)
					assert.True(t, ok)
					tocos = append(tocos, tc)
					return nil
				}))

				assert.EqualValues(t, 9, len(tocos))

				expectedTOCOs := []model.TOCO{
					model.TOCO{5, "m0", "t0", 100, beginTime.Add(time.Duration(1)*time.Second)},
					model.TOCO{5, "m0", "t0", 200, beginTime.Add(time.Duration(2)*time.Second)},
					model.TOCO{5, "m0", "t0", 300, beginTime.Add(time.Duration(3)*time.Second)},
					model.TOCO{6, "m5", "t2", 400, beginTime.Add(time.Duration(4)*time.Second)},
					model.TOCO{6, "m5", "t2", 500, beginTime.Add(time.Duration(5)*time.Second)},
					model.TOCO{2, "m2", "t2", 600, beginTime.Add(time.Duration(6)*time.Second)},
					model.TOCO{2, "m2", "t2", 700, beginTime.Add(time.Duration(7)*time.Second)},
					model.TOCO{7, "m1", "t5", 800, beginTime.Add(time.Duration(8)*time.Second)},
					model.TOCO{8, "m6", "t1", 900, beginTime.Add(time.Duration(9)*time.Second)},
				}

				for i, exp := range expectedTOCOs {
					assert.EqualValues(t, exp, *tocos[i])
				}
			},
		},
		{
			Name:    "認証エラー",
			Method:  http.MethodPost,
			Path:    "/ctg/heartrates",
			Token:   "invalidToken",
			Body:    test.CsvBody([][]string{}),
			Prepare: test.Prepares(truncate),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {

	})
}

/*
func TestCTG_UploadHeartRate(t *testing.T) {
	auth := (&F.CTGFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)
	influx := lib.GetInfluxDB()

	truncate := func(req *http.Request) {
		F.Truncate(db, "measurement", "patient", "measurement_terminal")
		influx.Delete("spiker", time.Unix(0, 0), time.Now(), "")
	}

	million := int64(1000000)

	beginTime := time.Now().Add(time.Duration(-1)*time.Hour)
	nano := int64(beginTime.UnixNano() / million) * million
	beginTime = time.Unix(0, nano).UTC()

	timestamp := func(sec int) string {
		return fmt.Sprintf("%d", int64(beginTime.Add(time.Duration(sec)*time.Second).UnixNano() / million))
	}

	httpTests := test.HttpTests{
		{
			Name:    "空アップロード",
			Method:  http.MethodPost,
			Path:    "/ctg/heartrates",
			Token:   auth.Token(3),
			Body:    test.CsvBody([][]string{}),
			Prepare: test.Prepares(truncate),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &uploadResponse{}).(*uploadResponse)

				assert.EqualValues(t, 0, res.Success)
				assert.EqualValues(t, 0, res.Failure)
			},
		},
		{
			Name:    "アップロード",
			Method:  http.MethodPost,
			Path:    "/ctg/heartrates",
			Token:   auth.Token(3),
			Body:    test.CsvBody([][]string{
				// 新規計測。端末も新規。先頭と末尾のタイムスタンプが記録される。
				[]string{"t0", "m0", "10", timestamp(1)},
				[]string{"t0", "m0", "30", timestamp(3)},
				[]string{"t0", "m0", "20", timestamp(2)},
				// 新規計測。端末は既存。
				[]string{"t2", "m5", "40", timestamp(4)},
				[]string{"t2", "m5", "50", timestamp(5)},
				// 既存の計測。末尾のタイムスタンプが更新される。
				[]string{"t2", "m2", "60", timestamp(6)},
				[]string{"t2", "m2", "70", timestamp(7)},
				// 新規計測だが同一コードの計測が別病院にある。
				[]string{"t5", "m1", "80", timestamp(8)},
				// 新規計測。端末も新規だが同一コードの端末が別病院にある。
				[]string{"t1", "m6", "90", timestamp(9)},
			}),
			Prepare: test.Prepares(truncate, func(req *http.Request) {
				// h - m  - t
				// 1 - m1 - t1
				// 2 - m2 - t2
				// 2 - m3 - t3
				// 3 - m4 - t4
				ts := []struct{ hid int; tc string }{
					{1, "t1"}, {2, "t2"}, {2, "t3"}, {3, "t4"},
				}

				terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 4, func(i int, r F.Record) {
					r["HospitalId"] = ts[i-1].hid
					r["Code"] = ts[i-1].tc
				}).([]*model.MeasurementTerminal)

				ms := []struct{ mc string; ti int }{
					{"m1", 0}, {"m2", 1}, {"m3", 2}, {"m4", 3},
				}

				patients := F.Insert(db, model.Patient{}, 0, 4, func(i int, r F.Record) {
					r["HospitalId"] = terminals[ms[i-1].ti].HospitalId
					r["Name"] = fmt.Sprintf("患者-%d", i)
				}).([]*model.Patient)

				F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
					r["Code"] = ms[i-1].mc
					r["PatientId"] = patients[i-1].Id
					r["TerminalId"] = terminals[ms[i-1].ti].Id
					r["FirstTime"] = beginTime
					r["LastTime"] = beginTime
				})
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &uploadResponse{}).(*uploadResponse)

				assert.EqualValues(t, 9, res.Success)
				assert.EqualValues(t, 0, res.Failure)

				// 計測端末t0,t5,t1がhospital=2に増える。
				assert.EqualValues(t, 7, F.Count(t, db, "measurement_terminal", nil))

				terminals := []*model.MeasurementTerminal{}
				F.Select(t, db, &terminals, F.NewQuery("").Asc("id"))
				assert.EqualValues(t, 7, len(terminals))

				expectTeminal := []struct{id int; code string; hid int}{
					{5, "t0", 2}, {6, "t5", 2}, {7, "t1", 2},
				}
				for i, exp := range expectTeminal {
					assert.EqualValues(t, exp.id, terminals[i+4].Id)
					assert.EqualValues(t, exp.code, terminals[i+4].Code)
					assert.EqualValues(t, exp.hid, terminals[i+4].HospitalId)
				}

				// 患者、計測が4つ増える。
				assert.EqualValues(t, 8, F.Count(t, db, "patient", nil))
				assert.EqualValues(t, 8, F.Count(t, db, "measurement", nil))

				patients := []*model.Patient{}
				measurements := []*model.Measurement{}

				F.Select(t, db, &patients, F.NewQuery("").Asc("id"))
				F.Select(t, db, &measurements, F.NewQuery("").Asc("id"))

				assert.EqualValues(t, 8, len(patients))
				assert.EqualValues(t, 8, len(measurements))

				expectPatient := []struct{id int; hid int}{
					{5, 2}, {6, 2}, {7, 2}, {8, 2},
				}
				for i, exp := range expectPatient {
					assert.EqualValues(t, exp.id, patients[i+4].Id)
					assert.EqualValues(t, exp.hid, patients[i+4].HospitalId)
				}

				expectMeasurement := []struct{id int; pid int; tid int; code string; b int; e int}{
					{5, 5, 5, "m0", 1, 3}, {6, 6, 2, "m5", 4, 5}, {7, 7, 6, "m1", 8, 8}, {8, 8, 7, "m6", 9, 9},
				}
				for i, exp := range expectMeasurement {
					assert.EqualValues(t, exp.id, measurements[i+4].Id)
					assert.EqualValues(t, exp.pid, measurements[i+4].PatientId)
					assert.EqualValues(t, exp.tid, measurements[i+4].TerminalId)
					assert.EqualValues(t, exp.code, measurements[i+4].Code)
					assert.EqualValues(t, beginTime.Add(time.Duration(exp.b)*time.Second).UnixNano(), measurements[i+4].FirstTime.UnixNano())
					assert.EqualValues(t, beginTime.Add(time.Duration(exp.e)*time.Second).UnixNano(), measurements[i+4].LastTime.UnixNano())
				}

				assert.EqualValues(t, beginTime.UnixNano(), measurements[1].FirstTime.UnixNano())
				assert.EqualValues(t, beginTime.Add(time.Duration(7)*time.Second).UnixNano(), measurements[1].LastTime.UnixNano())

				// InfluxDBに9レコード。
				points := []*model.HeartRate{}
				iq := `from(bucket:"spiker") |> range(start:-3h) |> group() |> sort(columns:["_time"])`
				influx.Select(iq, lib.PointConsumer(func(p lib.Point, f string) error {
					hr, ok := p.(*model.HeartRate)
					assert.True(t, ok)
					points = append(points, hr)
					return nil
				}))

				assert.EqualValues(t, 9, len(points))

				expectedPoints := []model.HeartRate{
					model.HeartRate{5, "m0", "t0", 10, beginTime.Add(time.Duration(1)*time.Second)},
					model.HeartRate{5, "m0", "t0", 20, beginTime.Add(time.Duration(2)*time.Second)},
					model.HeartRate{5, "m0", "t0", 30, beginTime.Add(time.Duration(3)*time.Second)},
					model.HeartRate{6, "m5", "t2", 40, beginTime.Add(time.Duration(4)*time.Second)},
					model.HeartRate{6, "m5", "t2", 50, beginTime.Add(time.Duration(5)*time.Second)},
					model.HeartRate{2, "m2", "t2", 60, beginTime.Add(time.Duration(6)*time.Second)},
					model.HeartRate{2, "m2", "t2", 70, beginTime.Add(time.Duration(7)*time.Second)},
					model.HeartRate{7, "m1", "t5", 80, beginTime.Add(time.Duration(8)*time.Second)},
					model.HeartRate{8, "m6", "t1", 90, beginTime.Add(time.Duration(9)*time.Second)},
				}

				for i, exp := range expectedPoints {
					assert.EqualValues(t, exp, *points[i])
				}
			},
		},
		{
			Name:    "認証エラー",
			Method:  http.MethodPost,
			Path:    "/ctg/heartrates",
			Token:   "invalidToken",
			Body:    test.CsvBody([][]string{}),
			Prepare: test.Prepares(truncate),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {

	})
}

func TestCTG_UploadTOCO(t *testing.T) {
	auth := (&F.CTGFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)
	influx := lib.GetInfluxDB()

	truncate := func(req *http.Request) {
		F.Truncate(db, "measurement", "patient", "measurement_terminal")
		influx.Delete("spiker", time.Unix(0, 0), time.Now(), "")
	}

	million := int64(1000000)

	beginTime := time.Now().Add(time.Duration(-1)*time.Hour)
	nano := int64(beginTime.UnixNano() / million) * million
	beginTime = time.Unix(0, nano).UTC()

	timestamp := func(sec int) string {
		return fmt.Sprintf("%d", int64(beginTime.Add(time.Duration(sec)*time.Second).UnixNano() / million))
	}

	httpTests := test.HttpTests{
		{
			Name:    "空アップロード",
			Method:  http.MethodPost,
			Path:    "/ctg/tocos",
			Token:   auth.Token(3),
			Body:    test.CsvBody([][]string{}),
			Prepare: test.Prepares(truncate),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &uploadResponse{}).(*uploadResponse)

				assert.EqualValues(t, 0, res.Success)
				assert.EqualValues(t, 0, res.Failure)
			},
		},
		{
			Name:    "アップロード",
			Method:  http.MethodPost,
			Path:    "/ctg/tocos",
			Token:   auth.Token(3),
			Body:    test.CsvBody([][]string{
				// 新規計測。端末も新規。先頭と末尾のタイムスタンプが記録される。
				[]string{"t0", "m0", "10", timestamp(1)},
				[]string{"t0", "m0", "30", timestamp(3)},
				[]string{"t0", "m0", "20", timestamp(2)},
				// 新規計測。端末は既存。
				[]string{"t2", "m5", "40", timestamp(4)},
				[]string{"t2", "m5", "50", timestamp(5)},
				// 既存の計測。末尾のタイムスタンプが更新される。
				[]string{"t2", "m2", "60", timestamp(6)},
				[]string{"t2", "m2", "70", timestamp(7)},
				// 新規計測だが同一コードの計測が別病院にある。
				[]string{"t5", "m1", "80", timestamp(8)},
				// 新規計測。端末も新規だが同一コードの端末が別病院にある。
				[]string{"t1", "m6", "90", timestamp(9)},
			}),
			Prepare: test.Prepares(truncate, func(req *http.Request) {
				// h - m  - t
				// 1 - m1 - t1
				// 2 - m2 - t2
				// 2 - m3 - t3
				// 3 - m4 - t4
				ts := []struct{ hid int; tc string }{
					{1, "t1"}, {2, "t2"}, {2, "t3"}, {3, "t4"},
				}

				terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 4, func(i int, r F.Record) {
					r["HospitalId"] = ts[i-1].hid
					r["Code"] = ts[i-1].tc
				}).([]*model.MeasurementTerminal)

				ms := []struct{ mc string; ti int }{
					{"m1", 0}, {"m2", 1}, {"m3", 2}, {"m4", 3},
				}

				patients := F.Insert(db, model.Patient{}, 0, 4, func(i int, r F.Record) {
					r["HospitalId"] = terminals[ms[i-1].ti].HospitalId
					r["Name"] = fmt.Sprintf("患者-%d", i)
				}).([]*model.Patient)

				F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
					r["Code"] = ms[i-1].mc
					r["PatientId"] = patients[i-1].Id
					r["TerminalId"] = terminals[ms[i-1].ti].Id
					r["FirstTime"] = beginTime
					r["LastTime"] = beginTime
				})
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &uploadResponse{}).(*uploadResponse)

				assert.EqualValues(t, 9, res.Success)
				assert.EqualValues(t, 0, res.Failure)

				// 計測端末t0,t5,t1がhospital=2に増える。
				assert.EqualValues(t, 7, F.Count(t, db, "measurement_terminal", nil))

				terminals := []*model.MeasurementTerminal{}
				F.Select(t, db, &terminals, F.NewQuery("").Asc("id"))
				assert.EqualValues(t, 7, len(terminals))

				expectTeminal := []struct{id int; code string; hid int}{
					{5, "t0", 2}, {6, "t5", 2}, {7, "t1", 2},
				}
				for i, exp := range expectTeminal {
					assert.EqualValues(t, exp.id, terminals[i+4].Id)
					assert.EqualValues(t, exp.code, terminals[i+4].Code)
					assert.EqualValues(t, exp.hid, terminals[i+4].HospitalId)
				}

				// 患者、計測が4つ増える。
				assert.EqualValues(t, 8, F.Count(t, db, "patient", nil))
				assert.EqualValues(t, 8, F.Count(t, db, "measurement", nil))

				patients := []*model.Patient{}
				measurements := []*model.Measurement{}

				F.Select(t, db, &patients, F.NewQuery("").Asc("id"))
				F.Select(t, db, &measurements, F.NewQuery("").Asc("id"))

				assert.EqualValues(t, 8, len(patients))
				assert.EqualValues(t, 8, len(measurements))

				expectPatient := []struct{id int; hid int}{
					{5, 2}, {6, 2}, {7, 2}, {8, 2},
				}
				for i, exp := range expectPatient {
					assert.EqualValues(t, exp.id, patients[i+4].Id)
					assert.EqualValues(t, exp.hid, patients[i+4].HospitalId)
				}

				expectMeasurement := []struct{id int; pid int; tid int; code string; b int; e int}{
					{5, 5, 5, "m0", 1, 3}, {6, 6, 2, "m5", 4, 5}, {7, 7, 6, "m1", 8, 8}, {8, 8, 7, "m6", 9, 9},
				}
				for i, exp := range expectMeasurement {
					assert.EqualValues(t, exp.id, measurements[i+4].Id)
					assert.EqualValues(t, exp.pid, measurements[i+4].PatientId)
					assert.EqualValues(t, exp.tid, measurements[i+4].TerminalId)
					assert.EqualValues(t, exp.code, measurements[i+4].Code)
					assert.EqualValues(t, beginTime.Add(time.Duration(exp.b)*time.Second).UnixNano(), measurements[i+4].FirstTime.UnixNano())
					assert.EqualValues(t, beginTime.Add(time.Duration(exp.e)*time.Second).UnixNano(), measurements[i+4].LastTime.UnixNano())
				}

				assert.EqualValues(t, beginTime.UnixNano(), measurements[1].FirstTime.UnixNano())
				assert.EqualValues(t, beginTime.Add(time.Duration(7)*time.Second).UnixNano(), measurements[1].LastTime.UnixNano())

				// InfluxDBに9レコード。
				points := []*model.TOCO{}
				iq := `from(bucket:"spiker") |> range(start:-3h) |> group() |> sort(columns:["_time"])`
				influx.Select(iq, lib.PointConsumer(func(p lib.Point, f string) error {
					hr, ok := p.(*model.TOCO)
					assert.True(t, ok)
					points = append(points, hr)
					return nil
				}))

				assert.EqualValues(t, 9, len(points))

				expectedPoints := []model.TOCO{
					model.TOCO{5, "m0", "t0", 10, beginTime.Add(time.Duration(1)*time.Second)},
					model.TOCO{5, "m0", "t0", 20, beginTime.Add(time.Duration(2)*time.Second)},
					model.TOCO{5, "m0", "t0", 30, beginTime.Add(time.Duration(3)*time.Second)},
					model.TOCO{6, "m5", "t2", 40, beginTime.Add(time.Duration(4)*time.Second)},
					model.TOCO{6, "m5", "t2", 50, beginTime.Add(time.Duration(5)*time.Second)},
					model.TOCO{2, "m2", "t2", 60, beginTime.Add(time.Duration(6)*time.Second)},
					model.TOCO{2, "m2", "t2", 70, beginTime.Add(time.Duration(7)*time.Second)},
					model.TOCO{7, "m1", "t5", 80, beginTime.Add(time.Duration(8)*time.Second)},
					model.TOCO{8, "m6", "t1", 90, beginTime.Add(time.Duration(9)*time.Second)},
				}

				for i, exp := range expectedPoints {
					assert.EqualValues(t, exp, *points[i])
				}
			},
		},
		{
			Name:    "認証エラー",
			Method:  http.MethodPost,
			Path:    "/ctg/tocos",
			Token:   "invalidToken",
			Body:    test.CsvBody([][]string{}),
			Prepare: test.Prepares(truncate),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {

	})
}
*/