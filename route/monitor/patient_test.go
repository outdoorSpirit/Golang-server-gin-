package monitor

import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestMonitorPatient_List(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Patient)

	beginTime := time.Now()

	httpTests := test.HttpTests{
		{
			Name:    "患者一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/patients",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listPatientsResponse{}).(*listPatientsResponse)

				assert.EqualValues(t, 7, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				assert.EqualValues(t, 7, len(res.Patients))

				expected := []int{1, 2, 3, 4, 5, 7, 9}
				for i, m := range res.Patients {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "範囲指定",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/patients",
			Query:   func(q url.Values) {
				q.Add("limit", "3")
				q.Add("offset", "2")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listPatientsResponse{}).(*listPatientsResponse)

				assert.EqualValues(t, 7, res.Total)
				assert.EqualValues(t, 3, res.Limit)
				assert.EqualValues(t, 2, res.Offset)

				assert.EqualValues(t, 3, len(res.Patients))

				expected := []int{3, 4, 5}
				for i, m := range res.Patients {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "期間絞り込み",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/patients",
			Query:   func(q url.Values) {
				q.Add("minutes", "3")
				q.Add("end", beginTime.Add(time.Duration(77)*time.Minute).Format(time.RFC3339))
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listPatientsResponse{}).(*listPatientsResponse)

				assert.EqualValues(t, 3, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				assert.EqualValues(t, 3, len(res.Patients))

				expected := []int{1, 2, 3}
				for i, m := range res.Patients {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "期間範囲を広げて確認",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/patients",
			Query:   func(q url.Values) {
				q.Add("minutes", "10")
				q.Add("end", beginTime.Add(time.Duration(77)*time.Minute).Format(time.RFC3339))
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listPatientsResponse{}).(*listPatientsResponse)

				assert.EqualValues(t, 4, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				assert.EqualValues(t, 4, len(res.Patients))

				expected := []int{1, 2, 3, 4}
				for i, m := range res.Patients {
					verify(expected[i], m)
				}
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement", "measurement_terminal", "patient")

		// h - p
		// 1 - [1,2,3,4,5,7,9]
		// 2 - [6,8,10]
		// 3 - []
		F.Insert(db, model.Patient{}, 0, 10, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 5 || i%2 == 1, 1, 2)
			r["Name"] = F.If(i <= 5, fmt.Sprintf("名前-%04d", i), nil)
		})

		F.Insert(db, model.MeasurementTerminal{}, 0, 1, func(i int, r F.Record) {
			r["HospitalId"] = 1
		})

		// p - m
		// 1 - -100~+100
		// 2 - -90~+90
		// ...
		// 6~ - x
		F.Insert(db, model.Measurement{}, 0, 5, func(i int, r F.Record) {
			r["PatientId"] = i
			r["TerminalId"] = 1
			r["FirstTime"] = beginTime.Add(-time.Duration(110-i*10)*time.Minute)
			r["LastTime"] = beginTime.Add(time.Duration(110-i*10)*time.Minute)
		})

		// 複数計測でレコードが増えない検証。
		F.Insert(db, model.Measurement{}, 0, 3, func(i int, r F.Record) {
			r["PatientId"] = 1
			r["TerminalId"] = 1
			r["FirstTime"] = beginTime.Add(-time.Duration(200-i*10)*time.Minute)
			r["LastTime"] = beginTime.Add(time.Duration(180-i*10)*time.Minute)
		})

		verify = func(id int, actual *model.Patient) {
			assert.EqualValues(t, id, actual.Id)

			// レスポンスに病院IDは含まれない。
			assert.EqualValues(t, 0, actual.HospitalId)

			if id <= 5 {
				assert.EqualValues(t, fmt.Sprintf("名前-%04d", id), *actual.Name)
			} else {
				assert.Nil(t, actual.Name)
			}
		}
	})
}

func TestMonitorPatient_Fetch(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Patient)

	httpTests := test.HttpTests{
		{
			Name:    "患者取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/patients/3",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Patient{}).(*model.Patient)

				verify(3, res)
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/patients/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/patients/6",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "patient")

		// h - p
		// 1 - [1,2,3,4,5,7,9]
		// 2 - [6,8,10]
		// 3 - []
		F.Insert(db, model.Patient{}, 0, 10, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 5 || i%2 == 1, 1, 2)
			r["Name"] = F.If(i <= 5, fmt.Sprintf("名前-%04d", i), nil)
		})

		verify = func(id int, actual *model.Patient) {
			assert.EqualValues(t, id, actual.Id)

			// レスポンスに病院IDは含まれない。
			assert.EqualValues(t, 0, actual.HospitalId)

			if id <= 5 {
				assert.EqualValues(t, fmt.Sprintf("名前-%04d", id), *actual.Name)
			} else {
				assert.Nil(t, actual.Name)
			}
		}
	})
}

func TestMonitorPatient_Update(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	testBody := func() map[string]interface{} {
		return map[string]interface{}{
			"name": "更新名",
			"age": 40,
			"numChildren": 50,
			"cesareanScar": false,
			"deliveryTime": 60,
			"bloodLoss": 70,
			"birthWeight": 80,
			"birthDatetime": "2021-01-02T03:04:05+09:00",
			"gestationalDays": 90,
			"apgarScore1min": 100,
			"apgarScore5min": 110,
			"umbilicalBlood": 120,
			"emergencyCesarean": false,
			"instrumentalLabor": false,
			"memo": "メモ",
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "患者更新",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/patients/1",
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Patient{}).(*model.Patient)

				assert.EqualValues(t, 1, res.Id)

				var actual *model.Patient
				if r, e := db.Get(model.Patient{}, 1); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, r)
					actual = r.(*model.Patient)
				}

				verify := func(patient *model.Patient) {
					assert.EqualValues(t, "更新名", *patient.Name)
					assert.EqualValues(t, 40, *patient.Age)
					assert.EqualValues(t, 50, *patient.NumChildren)
					assert.EqualValues(t, false, *patient.CesareanScar)
					assert.EqualValues(t, 60, *patient.DeliveryTime)
					assert.EqualValues(t, 70, *patient.BloodLoss)
					assert.EqualValues(t, 80, *patient.BirthWeight)
					assert.EqualValues(t,
						time.Date(2021, time.January, 2, 3, 4, 5, 0, time.UTC).Add(time.Duration(-9)*time.Hour).Unix(),
						patient.BirthDatetime.Unix(),
					)
					assert.EqualValues(t, 90, *patient.GestationalDays)
					assert.EqualValues(t, 100, *patient.ApgarScore1Min)
					assert.EqualValues(t, 110, *patient.ApgarScore5Min)
					assert.EqualValues(t, 120, *patient.UmbilicalBlood)
					assert.EqualValues(t, false, *patient.EmergencyCesarean)
					assert.EqualValues(t, false, *patient.InstrumentalLabor)
					assert.EqualValues(t, "メモ", patient.Memo)
				}

				verify(res)
				verify(actual)
			},
		},
		{
			Name:    "空で更新",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/patients/3",
			Body:    test.JsonBody(map[string]interface{}{
				"name": "",
				"age": 0,
				"numChildren": 0,
				"umbilicalBlood": nil,
				"emergencyCesarean": nil,
				"memo": "",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Patient{}).(*model.Patient)

				assert.EqualValues(t, 3, res.Id)

				var actual *model.Patient
				if r, e := db.Get(model.Patient{}, 3); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, r)
					actual = r.(*model.Patient)
				}

				verify := func(patient *model.Patient) {
					assert.EqualValues(t, "", *patient.Name)
					assert.EqualValues(t, 0, *patient.Age)
					assert.EqualValues(t, 0, *patient.NumChildren)
					assert.Nil(t, patient.CesareanScar)
					assert.Nil(t, patient.DeliveryTime)
					assert.Nil(t, patient.BloodLoss)
					assert.Nil(t, patient.BirthWeight)
					assert.Nil(t, patient.BirthDatetime)
					assert.Nil(t, patient.GestationalDays)
					assert.Nil(t, patient.ApgarScore1Min)
					assert.Nil(t, patient.ApgarScore5Min)
					assert.Nil(t, patient.UmbilicalBlood)
					assert.Nil(t, patient.EmergencyCesarean)
					assert.Nil(t, patient.InstrumentalLabor)
					assert.EqualValues(t, "", patient.Memo)
				}

				verify(res)
				verify(actual)
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/patients/0",
			Body:    test.JsonBody(map[string]interface{}{
				"name": "更新名",
				"memo": "メモ",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/patients/2",
			Body:    test.JsonBody(map[string]interface{}{
				"name": "更新名",
				"memo": "メモ",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	validation := func(title string, key string, value interface{}) test.HttpTest {
		body := testBody()

		body[key] = value

		return test.HttpTest {
			Name:    title,
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/patients/5",
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("nameが長い", "name", strings.Repeat("a", 65)),
		validation("ageが負", "age", -1),
		validation("numChildrenが負", "numChildren", -1),
		validation("deliveryTimeが負", "deliveryTime", -1),
		validation("bloodLossが負", "bloodLoss", -1),
		validation("birthWeightが負", "birthWeight", -1),
		validation("birthDatetimeが不正", "birthDatetime", "2021/1/2 3:04:5"),
		validation("gestationalDaysが負", "gestationalDays", -1),
		validation("umbilicalBloodが負", "umbilicalBlood", -1),
		validation("memoが長い", "memo", strings.Repeat("0123456789", 200) + "a"),
	)

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "patient")

		// h - p
		// 1 - [1,3,5]
		// 2 - [2,4]
		// 3 - []
		F.Insert(db, model.Patient{}, 0, 5, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 2)
			r["Name"] = F.If(i <= 3, fmt.Sprintf("名前-%04d", i), nil)
			r["Age"] = F.If(i <= 3, 4, nil)
			r["NumChildren"] = F.If(i <= 3, 5, nil)
			r["CesareanScar"] = F.If(i <= 3, true, nil)
			r["DeliveryTime"] = F.If(i <= 3, 6, nil)
			r["BloodLoss"] = F.If(i <= 3, 7, nil)
			r["BirthWeight"] = F.If(i <= 3, 8, nil)
			r["BirthDatetime"] = F.If(i <= 3, time.Now(), nil)
			r["GestationalDays"] = F.If(i <= 3, 9, nil)
			r["ApgarScore1Min"] = F.If(i <= 3, 10, nil)
			r["ApgarScore5Min"] = F.If(i <= 3, 11, nil)
			r["UmbilicalBlood"] = F.If(i <= 3, 12, nil)
			r["EmergencyCesarean"] = F.If(i <= 3, true, nil)
			r["InstrumentalLabor"] = F.If(i <= 3, true, nil)
		})
	})
}