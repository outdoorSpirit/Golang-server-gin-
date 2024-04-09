package annotation

import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	//"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestAnnotationPatient_List(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Patient)

	beginTime := time.Now()

	httpTests := test.HttpTests{
		{
			Name:    "患者一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/patients",
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
			Path:    "/annotation/hospitals/hospital-0001/patients",
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
			Path:    "/annotation/hospitals/hospital-0001/patients",
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
			Path:    "/annotation/hospitals/hospital-0001/patients",
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
		F.Truncate(db, "measurement", "measurement_terminal", "patient", "hospital")

		hospitals := F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		}).([]*model.Hospital)

		// h - p
		// 1 - [1,2,3,4,5,7,9]
		// 2 - [6,8,10]
		// 3 - []
		F.Insert(db, model.Patient{}, 0, 10, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 5 || i%2 == 1, hospitals[0].Id, hospitals[1].Id)
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

func TestAnnotationPatient_Fetch(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Patient)

	httpTests := test.HttpTests{
		{
			Name:    "患者取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/patients/3",
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
			Path:    "/annotation/hospitals/hospital-0001/patients/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/patients/6",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "patient", "hospital")

		hospitals := F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		}).([]*model.Hospital)

		// h - p
		// 1 - [1,2,3,4,5,7,9]
		// 2 - [6,8,10]
		// 3 - []
		F.Insert(db, model.Patient{}, 0, 10, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 5 || i%2 == 1, hospitals[0].Id, hospitals[1].Id)
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