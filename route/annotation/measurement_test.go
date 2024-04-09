package annotation

import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestAnnotationMeasurement_List(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.MeasurementEntity)

	httpTests := test.HttpTests{
		{
			Name:    "計測一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listMeasurementsResponse{}).(*listMeasurementsResponse)

				assert.EqualValues(t, 9, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				assert.EqualValues(t, 9, len(res.Measurements))

				expected := []int{5, 3, 2, 1, 6, 7, 8, 9, 10}
				for i, m := range res.Measurements {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "患者指定",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements",
			Query:   func(q url.Values) {
				q.Add("patient", "1")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listMeasurementsResponse{}).(*listMeasurementsResponse)

				assert.EqualValues(t, 7, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				assert.EqualValues(t, 7, len(res.Measurements))

				expected := []int{2, 1, 6, 7, 8, 9, 10}
				for i, m := range res.Measurements {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "機器指定",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements",
			Query:   func(q url.Values) {
				q.Add("terminal", "2")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listMeasurementsResponse{}).(*listMeasurementsResponse)

				assert.EqualValues(t, 7, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				assert.EqualValues(t, 7, len(res.Measurements))

				expected := []int{3, 1, 6, 7, 8, 9, 10}
				for i, m := range res.Measurements {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "範囲指定",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements",
			Query:   func(q url.Values) {
				q.Add("limit", "3")
				q.Add("offset", "2")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listMeasurementsResponse{}).(*listMeasurementsResponse)

				assert.EqualValues(t, 9, res.Total)
				assert.EqualValues(t, 3, res.Limit)
				assert.EqualValues(t, 2, res.Offset)

				assert.EqualValues(t, 3, len(res.Measurements))

				expected := []int{2, 1, 6}
				for i, m := range res.Measurements {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement_terminal", "patient", "measurement", "hospital")

		hospitals := F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		}).([]*model.Hospital)

		// h - p
		// 1 - [1,2,3]
		// 2 - [4]
		patients := F.Insert(db, model.Patient{}, 0, 4, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 3, hospitals[0].Id, hospitals[1].Id)
		}).([]*model.Patient)

		// h - t
		// 1 - [2,3,4]
		// 2 - [1]
		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 4, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i >= 2, hospitals[0].Id, hospitals[1].Id)
		}).([]*model.MeasurementTerminal)

		// m     - p - t
		// 1     - 1 - 2
		// 2     - 1 - 3
		// 3     - 2 - 2
		// 4     - 4 - 1 -- hospital2
		// 5     - 3 - 4
		// 6..10 - 1 - 2
		ms := []struct{pi int; ti int; t int}{
			{1, 2, 1}, {1, 3, 2}, {2, 2, 3}, {4, 1, 4}, {3, 4, 5},
			{1, 2, -1}, {1, 2, -2}, {1, 2, -3}, {1, 2, -4}, {1, 2, -5},
		}
		now := time.Now()
		F.Insert(db, model.Measurement{}, 0, len(ms), func(i int, r F.Record) {
			r["Code"] = fmt.Sprintf("m%04d", i)
			r["PatientId"] = patients[ms[i-1].pi-1].Id
			r["TerminalId"] = terminals[ms[i-1].ti-1].Id
			r["CreatedAt"] = now.Add(time.Duration(ms[i-1].t)*time.Hour)
		})

		verify = func(id int, actual *model.MeasurementEntity) {
			assert.EqualValues(t, id, actual.Id)
			assert.EqualValues(t, fmt.Sprintf("m%04d", id), actual.Code)
			assert.EqualValues(t, patients[ms[id-1].pi-1].Id, actual.Patient.Id)
			assert.EqualValues(t, terminals[ms[id-1].ti-1].Id, actual.Terminal.Id)
		}
	})
}

func TestAnnotationMeasurement_Fetch(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.MeasurementEntity)

	httpTests := test.HttpTests{
		{
			Name:    "計測取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/3",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.MeasurementEntity{}).(*model.MeasurementEntity)

				verify(3, res)
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/4",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement_terminal", "patient", "measurement", "hospital")

		hospitals := F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		}).([]*model.Hospital)

		// h - p
		// 1 - [1,2,3]
		// 2 - [4]
		patients := F.Insert(db, model.Patient{}, 0, 4, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 3, hospitals[0].Id, hospitals[1].Id)
		}).([]*model.Patient)

		// h - t
		// 1 - [2,3,4]
		// 2 - [1]
		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 4, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i >= 2, hospitals[0].Id, hospitals[1].Id)
		}).([]*model.MeasurementTerminal)

		// m     - p - t
		// 1     - 1 - 2
		// 2     - 1 - 3
		// 3     - 2 - 2
		// 4     - 4 - 1 -- hospital2
		// 5     - 3 - 4
		// 6..10 - 1 - 2
		ms := []struct{pi int; ti int; t int}{
			{1, 2, 1}, {1, 3, 2}, {2, 2, 3}, {4, 1, 4}, {3, 4, 5},
			{1, 2, -1}, {1, 2, -2}, {1, 2, -3}, {1, 2, -4}, {1, 2, -5},
		}
		now := time.Now()
		F.Insert(db, model.Measurement{}, 0, len(ms), func(i int, r F.Record) {
			r["Code"] = fmt.Sprintf("m%04d", i)
			r["PatientId"] = patients[ms[i-1].pi-1].Id
			r["TerminalId"] = terminals[ms[i-1].ti-1].Id
			r["CreatedAt"] = now.Add(time.Duration(ms[i-1].t)*time.Hour)
		})

		verify = func(id int, actual *model.MeasurementEntity) {
			assert.EqualValues(t, id, actual.Id)
			assert.EqualValues(t, fmt.Sprintf("m%04d", id), actual.Code)
			assert.EqualValues(t, patients[ms[id-1].pi-1].Id, actual.Patient.Id)
			assert.EqualValues(t, terminals[ms[id-1].ti-1].Id, actual.Terminal.Id)
		}
	})
}

func TestAnnotationMeasurement_GetSilent(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	now := time.Now().In(time.UTC)

	httpTests := test.HttpTests{
		{
			Name:    "サイレント中",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/silent",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &getSilentStateResponse{}).(*getSilentStateResponse)

				assert.True(t, res.IsSilent)
				assert.EqualValues(t, now.Add(time.Duration(-1)*time.Minute).Unix(), res.Parameters.SilentFrom.Unix())
				assert.EqualValues(t, now.Add(time.Duration(1)*time.Minute).Unix(), res.Parameters.SilentUntil.Unix())
			},
		},
		{
			Name:    "サイレント中ではない",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/3/silent",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &getSilentStateResponse{}).(*getSilentStateResponse)

				assert.False(t, res.IsSilent)
				assert.Nil(t, res.Parameters)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/2/silent",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement_terminal", "patient", "measurement", "hospital")

		F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		})

		F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = i
		})

		F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = 3
		})

		F.Insert(db, model.Measurement{}, 0, 5, func(i int, r F.Record) {
			r["Code"] = fmt.Sprintf("m%04d", i)
			r["PatientId"] = F.If(i <= 3, 1, F.If(i <= 4, 2, 3))
			r["TerminalId"] = F.If(i <= 3, 1, F.If(i <= 4, 2, 3))
		})

		mas := []struct{ mid int; from int }{
			{1, -5}, {1, 1},
			{2, -1}, {2, 3},
			{3, 1}, {3, 5},
			{4, -1}, {4, 3},
		}
		F.Insert(db, model.MeasurementAlert{}, 0, len(mas), func(i int, r F.Record) {
			r["MeasurementId"] = mas[i-1].mid
			r["SilentFrom"] = now.Add(time.Duration(mas[i-1].from)*time.Minute)
			r["SilentUntil"] = now.Add(time.Duration(mas[i-1].from+2)*time.Minute)
		})
	})
}
