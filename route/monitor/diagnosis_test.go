package monitor

/*
import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	//"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	//C "github.com/spiker/spiker-server/constant"
	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestMonitorDiagnosis_List(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Now()

	var verify func(int, *model.DiagnosisEntity)

	httpTests := test.HttpTests{
		{
			Name:    "診断一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/diagnoses",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listDiagnosesResponse{}).(*listDiagnosesResponse)

				assert.EqualValues(t, 3, len(res.Diagnoses))

				expected := []int{5, 4, 3}
				for i, m := range res.Diagnoses {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "計測が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/0/diagnoses",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/diagnoses",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "diagnosis_algorithm", "diagnosis_content", "diagnosis", "measurement_terminal", "patient", "measurement")

		patients := F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = i
		}).([]*model.Patient)

		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = i
		}).([]*model.MeasurementTerminal)

		// h - m
		// 1 - [1,2,3]
		// 2 - [4]
		// 3 - []
		measurements := F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
			r["Code"] = fmt.Sprintf("measurement-%04d", i)
			r["PatientId"] = F.If(i <= 3, patients[0].Id, patients[1].Id)
			r["TerminalId"] = F.If(i <= 3, terminals[0].Id, terminals[1].Id)
		}).([]*model.Measurement)

		// m - d
		// 1 - [1(s),2(w)]
		// 2 - [3(d),4(s),5(w)]
		// 3 - [6(d)]
		// 4 - [7(s),8(d)]
		m2d := []struct{mi int; di int}{
			{1, 1}, {1, 2},
			{2, 3}, {2, 4}, {2, 5},
			{3, 6},
			{4, 7}, {4, 8},
		}
		now := time.Now()
		diagnoses := F.Insert(db, model.Diagnosis{}, 0, 8, func(i int, r F.Record) {
			r["MeasurementId"] = measurements[m2d[i-1].mi-1].Id
			r["CreatedAt"] = now.Add(time.Duration(i)*time.Hour)
		}).([]*model.Diagnosis)

		// d - a
		// 1 - 1
		// 2 - 2
		// 3 - 1
		// ...
		algorithms := F.Insert(db, model.DiagnosisAlgorithm{}, 0, 2, func(i int, r F.Record) {
			r["Name"] = fmt.Sprintf("algorithm-%04d", i)
		}).([]*model.DiagnosisAlgorithm)

		F.Insert(db, model.ComputedDiagnosis{}, 0, 8, func(i int, r F.Record) {
			r["DiagnosisId"] = diagnoses[i-1].Id
			r["AlgorithmId"] = algorithms[(i-1)%2].Id
		})

		// d - c
		// 1 - [1]
		// 2 - [2]
		// 3 - [3,4,5]
		// 4 - [6,7]
		// 5 - []
		// 6~8 - []
		d2c := []struct{ did int; cid int }{
			{1, 1},
			{2, 2},
			{3, 3}, {3, 4}, {3, 5},
			{4, 6}, {4, 7},
		}

		contents := F.Insert(db, model.DiagnosisContent{}, 0, len(d2c), func(i int, r F.Record) {
			r["DiagnosisId"] = diagnoses[d2c[i-1].did-1].Id
			r["Parameters"] = model.JSON(fmt.Sprintf(`{"risk":%d}`, i))
			r["RangeFrom"] = beginTime.Add(time.Duration(i)*time.Minute)
		}).([]*model.DiagnosisContent)

		verify = func(id int, actual *model.DiagnosisEntity) {
			assert.EqualValues(t, id, actual.Id)

			expectedContents := []*model.DiagnosisContent{}
			for _, d := range d2c {
				if d.did == id {
					expectedContents = append(expectedContents, contents[d.cid-1])
				}
			}

			assert.EqualValues(t, len(expectedContents), len(actual.Contents))
			for i, c := range actual.Contents {
				exp := expectedContents[i]

				assert.EqualValues(t, exp.Id, c.Id)

				if parameters, e := c.Parameters.UnmarshalObject(); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.EqualValues(t, map[string]interface{}{"risk": float64(exp.Id)}, parameters)
				}
			}

			algId := (id-1)%2+1
			assert.EqualValues(t, algId, actual.Algorithm.Id)
			assert.EqualValues(t, fmt.Sprintf("algorithm-%04d", algId), actual.Algorithm.Name)
		}
	})
}

func TestMonitorDiagnosis_Fetch(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Now()

	var verify func(int, *model.DiagnosisEntity)

	httpTests := test.HttpTests{
		{
			Name:    "診断取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/diagnoses/4",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.DiagnosisEntity{}).(*model.DiagnosisEntity)

				verify(4, res)
			},
		},
		{
			Name:    "項目が空の診断",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/diagnoses/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.DiagnosisEntity{}).(*model.DiagnosisEntity)

				verify(5, res)
				assert.EqualValues(t, 0, len(res.Contents))
			},
		},
		{
			Name:    "診断が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/diagnoses/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測との組み合わせが違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/diagnoses/4",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/diagnoses/7",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "diagnosis_algorithm", "diagnosis", "measurement_terminal", "patient", "measurement")

		patients := F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = i
		}).([]*model.Patient)

		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = i
		}).([]*model.MeasurementTerminal)

		// h - m
		// 1 - [1,2,3]
		// 2 - [4]
		// 3 - []
		measurements := F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
			r["Code"] = fmt.Sprintf("measurement-%04d", i)
			r["PatientId"] = F.If(i <= 3, patients[0].Id, patients[1].Id)
			r["TerminalId"] = F.If(i <= 3, terminals[0].Id, terminals[1].Id)
		}).([]*model.Measurement)

		// m - d
		// 1 - [1(s),2(w)]
		// 2 - [3(d),4(s),5(w)]
		// 3 - [6(d)]
		// 4 - [7(s),8(d)]
		m2d := []struct{mi int; di int}{
			{1, 1}, {1, 2},
			{2, 3}, {2, 4}, {2, 5},
			{3, 6},
			{4, 7}, {4, 8},
		}
		now := time.Now()
		diagnoses := F.Insert(db, model.Diagnosis{}, 0, 8, func(i int, r F.Record) {
			r["MeasurementId"] = measurements[m2d[i-1].mi-1].Id
			r["CreatedAt"] = now.Add(time.Duration(i)*time.Hour)
		}).([]*model.Diagnosis)

		// d - a
		// 1 - 1
		// 2 - 2
		// 3 - 1
		// ...
		algorithms := F.Insert(db, model.DiagnosisAlgorithm{}, 0, 2, func(i int, r F.Record) {
			r["Name"] = fmt.Sprintf("algorithm-%04d", i)
		}).([]*model.DiagnosisAlgorithm)

		F.Insert(db, model.ComputedDiagnosis{}, 0, 8, func(i int, r F.Record) {
			r["DiagnosisId"] = diagnoses[i-1].Id
			r["AlgorithmId"] = algorithms[(i-1)%2].Id
		})

		// d - c
		// 1 - [1]
		// 2 - [2]
		// 3 - [3,4,5]
		// 4 - [6,7]
		// 5 - []
		// 6~8 - []
		d2c := []struct{ did int; cid int }{
			{1, 1},
			{2, 2},
			{3, 3}, {3, 4}, {3, 5},
			{4, 6}, {4, 7},
		}

		contents := F.Insert(db, model.DiagnosisContent{}, 0, len(d2c), func(i int, r F.Record) {
			r["DiagnosisId"] = diagnoses[d2c[i-1].did-1].Id
			r["Parameters"] = model.JSON(fmt.Sprintf(`{"risk":%d}`, i))
			r["RangeFrom"] = beginTime.Add(time.Duration(i)*time.Minute)
		}).([]*model.DiagnosisContent)

		verify = func(id int, actual *model.DiagnosisEntity) {
			assert.EqualValues(t, id, actual.Id)

			expectedContents := []*model.DiagnosisContent{}
			for _, d := range d2c {
				if d.did == id {
					expectedContents = append(expectedContents, contents[d.cid-1])
				}
			}

			assert.EqualValues(t, len(expectedContents), len(actual.Contents))
			for i, c := range actual.Contents {
				exp := expectedContents[i]

				assert.EqualValues(t, exp.Id, c.Id)

				if parameters, e := c.Parameters.UnmarshalObject(); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.EqualValues(t, map[string]interface{}{"risk": float64(exp.Id)}, parameters)
				}
			}

			algId := (id-1)%2+1
			assert.EqualValues(t, algId, actual.Algorithm.Id)
			assert.EqualValues(t, fmt.Sprintf("algorithm-%04d", algId), actual.Algorithm.Name)
		}
	})
}

func TestMonitorDiagnosis_Update(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Now()

	var verify func(int, *model.DiagnosisEntity)

	testBody := func() map[string]interface{} {
		return map[string]interface{}{
			"memo": "メモ",
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "診断更新",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/diagnoses/4",
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.DiagnosisEntity{}).(*model.DiagnosisEntity)

				verify(4, res)

				var actual *model.Diagnosis
				if r, e := db.Get(model.Diagnosis{}, 4); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, r)
					actual = r.(*model.Diagnosis)
				}

				assert.EqualValues(t, "メモ", actual.Memo)
			},
		},
		{
			Name:    "診断が無い",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/diagnoses/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測との組み合わせが違う",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/diagnoses/4",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/diagnoses/7",
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
			Path:    "/1/measurements/2/diagnoses/3",
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("memoが長い", "memo", strings.Repeat("0123456789", 200) + "a"),
	)

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "diagnosis_algorithm", "diagnosis", "measurement_terminal", "patient", "measurement")

		patients := F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = i
		}).([]*model.Patient)

		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = i
		}).([]*model.MeasurementTerminal)

		// h - m
		// 1 - [1,2,3]
		// 2 - [4]
		// 3 - []
		measurements := F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
			r["Code"] = fmt.Sprintf("measurement-%04d", i)
			r["PatientId"] = F.If(i <= 3, patients[0].Id, patients[1].Id)
			r["TerminalId"] = F.If(i <= 3, terminals[0].Id, terminals[1].Id)
		}).([]*model.Measurement)

		// m - d
		// 1 - [1(s),2(w)]
		// 2 - [3(d),4(s),5(w)]
		// 3 - [6(d)]
		// 4 - [7(s),8(d)]
		m2d := []struct{mi int; di int}{
			{1, 1}, {1, 2},
			{2, 3}, {2, 4}, {2, 5},
			{3, 6},
			{4, 7}, {4, 8},
		}
		now := time.Now()
		diagnoses := F.Insert(db, model.Diagnosis{}, 0, 8, func(i int, r F.Record) {
			r["MeasurementId"] = measurements[m2d[i-1].mi-1].Id
			r["CreatedAt"] = now.Add(time.Duration(i)*time.Hour)
		}).([]*model.Diagnosis)

		// d - a
		// 1 - 1
		// 2 - 2
		// 3 - 1
		// ...
		algorithms := F.Insert(db, model.DiagnosisAlgorithm{}, 0, 2, func(i int, r F.Record) {
			r["Name"] = fmt.Sprintf("algorithm-%04d", i)
		}).([]*model.DiagnosisAlgorithm)

		F.Insert(db, model.ComputedDiagnosis{}, 0, 8, func(i int, r F.Record) {
			r["DiagnosisId"] = diagnoses[i-1].Id
			r["AlgorithmId"] = algorithms[(i-1)%2].Id
		})

		// d - c
		// 1 - [1]
		// 2 - [2]
		// 3 - [3,4,5]
		// 4 - [6,7]
		// 5 - []
		// 6~8 - []
		d2c := []struct{ did int; cid int }{
			{1, 1},
			{2, 2},
			{3, 3}, {3, 4}, {3, 5},
			{4, 6}, {4, 7},
		}

		contents := F.Insert(db, model.DiagnosisContent{}, 0, len(d2c), func(i int, r F.Record) {
			r["DiagnosisId"] = diagnoses[d2c[i-1].did-1].Id
			r["Parameters"] = model.JSON(fmt.Sprintf(`{"risk":%d}`, i))
			r["RangeFrom"] = beginTime.Add(time.Duration(i)*time.Minute)
		}).([]*model.DiagnosisContent)

		verify = func(id int, actual *model.DiagnosisEntity) {
			assert.EqualValues(t, id, actual.Id)

			expectedContents := []*model.DiagnosisContent{}
			for _, d := range d2c {
				if d.did == id {
					expectedContents = append(expectedContents, contents[d.cid-1])
				}
			}

			assert.EqualValues(t, len(expectedContents), len(actual.Contents))
			for i, c := range actual.Contents {
				exp := expectedContents[i]

				assert.EqualValues(t, exp.Id, c.Id)

				if parameters, e := c.Parameters.UnmarshalObject(); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.EqualValues(t, map[string]interface{}{"risk": float64(exp.Id)}, parameters)
				}
			}

			algId := (id-1)%2+1
			assert.EqualValues(t, algId, actual.Algorithm.Id)
			assert.EqualValues(t, fmt.Sprintf("algorithm-%04d", algId), actual.Algorithm.Name)
		}
	})
}
*/