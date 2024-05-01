package annotation

/*
import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	//"net/url"
	//"strings"
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

func TestAnnotationDiagnosis_List(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Now()

	var verify func(int, *model.DiagnosisEntity)

	httpTests := test.HttpTests{
		{
			Name:    "診断一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/diagnoses",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/0/diagnoses",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/2/diagnoses",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/4/diagnoses",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "diagnosis_algorithm", "diagnosis_content", "diagnosis", "measurement_terminal", "patient", "measurement", "hospital")

		hospitals := F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		}).([]*model.Hospital)

		patients := F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = hospitals[i-1].Id
		}).([]*model.Patient)

		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = hospitals[i-1].Id
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

func TestAnnotationDiagnosis_Fetch(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Now()

	var verify func(int, *model.DiagnosisEntity)

	httpTests := test.HttpTests{
		{
			Name:    "診断取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/diagnoses/4",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/diagnoses/5",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/diagnoses/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/2/diagnoses/4",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測との組み合わせが違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/diagnoses/4",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/4/diagnoses/7",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "diagnosis_algorithm", "diagnosis", "measurement_terminal", "patient", "measurement", "hospital")

		hospitals := F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		}).([]*model.Hospital)

		patients := F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = hospitals[i-1].Id
		}).([]*model.Patient)

		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = hospitals[i-1].Id
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

func TestAnnotationDiagnosis_Register(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	verifyDiagnosis := func(m *model.Diagnosis, fromDB bool, id, mid, bpm, risk, fh, fm, uh, um int, memo string) {
		if fromDB {
			assert.EqualValues(t, id, m.Id)
		}
		if fromDB {
			assert.EqualValues(t, mid, m.MeasurementId)
		}
		if bpm < 0 {
			assert.Nil(t, m.BaselineBpm)
		} else {
			assert.EqualValues(t, bpm, *m.BaselineBpm)
		}
		if risk < 0 {
			assert.Nil(t, m.MaximumRisk)
		} else {
			assert.EqualValues(t, risk, *m.MaximumRisk)
		}
		assert.EqualValues(t, time.Date(2021, time.October, 1, fh, fm, 0, 0, time.UTC), m.RangeFrom)
		assert.EqualValues(t, time.Date(2021, time.October, 1, uh, um, 0, 0, time.UTC), m.RangeUntil)
		assert.EqualValues(t, memo, m.Memo)
	}

	verifyContent := func(m *model.DiagnosisContent, fromDB bool, id, did, risk, fh, fm, uh, um int, memo string, params map[string]interface{}) {
		if fromDB {
			assert.EqualValues(t, id, m.Id)
		}
		if fromDB {
			assert.EqualValues(t, did, m.DiagnosisId)
		}
		if risk < 0 {
			assert.Nil(t, m.Risk)
		} else {
			assert.EqualValues(t, risk, *m.Risk)
		}
		assert.EqualValues(t, time.Date(2021, time.October, 1, fh, fm, 0, 0, time.UTC), m.RangeFrom)
		assert.EqualValues(t, time.Date(2021, time.October, 1, uh, um, 0, 0, time.UTC), m.RangeUntil)
		if ps, e := m.Parameters.UnmarshalObject(); e != nil {
			assert.FailNow(t, e.Error())
		} else {
			assert.EqualValues(t, params, ps)
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "空の状態から登録",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/diagnoses",
			Body:    test.JsonBody(map[string]interface{}{
				"contents": []map[string]interface{}{
					map[string]interface{}{
						"range_from": "2021-10-01T12:00:00+00:00",
						"range_until": "2021-10-01T12:10:00+00:00",
						"parameters": map[string]interface{}{
							"Acceleration": nil,
						},
						"memo": "メモ-1",
					},
					map[string]interface{}{
						"range_from": "2021-10-01T12:20:00+00:00",
						"range_until": "2021-10-01T12:30:00+00:00",
						"parameters": map[string]interface{}{
							"Baseline-NORMAL": 100,
							"BaselineVariability-DECREASE": 30,
						},
						"memo": "メモ-2",
					},
					map[string]interface{}{
						"range_from": "2021-10-01T12:40:00+00:00",
						"range_until": "2021-10-01T12:50:00+00:00",
						"parameters": map[string]interface{}{
							"Deceleration-HI_LD": nil,
						},
						"memo": "メモ-3",
					},
					map[string]interface{}{
						"range_from": "2021-10-01T13:00:00+00:00",
						"range_until": "2021-10-01T13:10:00+00:00",
						"parameters": map[string]interface{}{
							"Baseline-DECELERATION": 120,
							"BaselineVariability-INCREASE": 50,
						},
						"memo": "メモ-4",
					},
					map[string]interface{}{
						"range_from": "2021-10-01T13:20:00+00:00",
						"range_until": "2021-10-01T13:30:00+00:00",
						"parameters": map[string]interface{}{
							"Deceleration-HI_LD": nil,
						},
						"memo": "メモ-5",
					},
				},
				"memo": "メモ",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.DiagnosisEntity{}).(*model.DiagnosisEntity)

				fmt.Println(res)

				assert.Nil(t, res.Algorithm)
				assert.EqualValues(t, 5, len(res.Contents))

				verifyDiagnosis(res.Diagnosis, false, 1, 2, 120, 4, 12, 0, 13, 30, "メモ")

				verify := func(cs []*model.DiagnosisContent, fromDB bool) {
					verifyContent(cs[0], fromDB, 1, 1, -1, 12, 0, 12, 10, "メモ-1", map[string]interface{}{
						"Acceleration": nil,
					})
					verifyContent(cs[1], fromDB, 2, 1, 2, 12, 20, 12, 30, "メモ-2", map[string]interface{}{
						"Baseline-NORMAL": float64(100),
						"BaselineVariability-DECREASE": float64(30),
					})
					verifyContent(cs[2], fromDB, 3, 1, 4, 12, 40, 12, 50, "メモ-3", map[string]interface{}{
						"Deceleration-HI_LD": nil,
					})
					verifyContent(cs[3], fromDB, 4, 1, 2, 13, 0, 13, 10, "メモ-4", map[string]interface{}{
						"Baseline-DECELERATION": float64(120),
						"BaselineVariability-INCREASE": float64(50),
					})
					verifyContent(cs[4], fromDB, 5, 1, 3, 13, 20, 13, 30, "メモ-5", map[string]interface{}{
						"Deceleration-HI_LD": nil,
					})
				}

				if actual, e := db.Get(model.Diagnosis{}, res.Diagnosis.Id); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					verifyDiagnosis(actual.(*model.Diagnosis), true, 1, 2, 120, 4, 12, 0, 13, 30, "メモ")
				}

				verify(res.Contents, false)

				actualContents := []*model.DiagnosisContent{}
				F.Select(t, db, &actualContents, F.NewQuery(
					"id IN ($1, $2, $3, $4, $5)",
					res.Contents[0].Id, res.Contents[1].Id, res.Contents[2].Id, res.Contents[3].Id, res.Contents[4].Id,
				).Asc("id"))

				verify(actualContents, true)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "diagnosis_algorithm", "diagnosis_content", "diagnosis", "measurement_terminal", "patient", "measurement", "hospital")

		hospitals := F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		}).([]*model.Hospital)

		patients := F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = hospitals[i-1].Id
		}).([]*model.Patient)

		terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = hospitals[i-1].Id
		}).([]*model.MeasurementTerminal)

		// h - m
		// 1 - [1,2,3]
		// 2 - [4]
		// 3 - []
		F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
			r["Code"] = fmt.Sprintf("measurement-%04d", i)
			r["PatientId"] = F.If(i <= 3, patients[0].Id, patients[1].Id)
			r["TerminalId"] = F.If(i <= 3, terminals[0].Id, terminals[1].Id)
		})
	})
}
*/