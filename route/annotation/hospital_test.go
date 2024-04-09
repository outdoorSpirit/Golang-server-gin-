package annotation

import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	//"time"

	"github.com/stretchr/testify/assert"

	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestAnnotationHospital_List(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Hospital)

	httpTests := test.HttpTests{
		{
			Name:    "一覧",
			Method:  http.MethodGet,
			Path:    "/annotation/hospitals",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listHospitalsResponse{}).(*listHospitalsResponse)

				assert.EqualValues(t, 10, len(res.Hospitals))
				assert.EqualValues(t, 10, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				for i, m := range res.Hospitals {
					verify(i+1, m)
				}
			},
		},
		{
			Name:    "範囲指定",
			Method:  http.MethodGet,
			Path:    "/annotation/hospitals",
			Token:   auth.Token(1),
			Query:   func(q url.Values) {
				q.Add("limit", "3")
				q.Add("offset", "2")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listHospitalsResponse{}).(*listHospitalsResponse)

				assert.EqualValues(t, 3, len(res.Hospitals))
				assert.EqualValues(t, 10, res.Total)
				assert.EqualValues(t, 3, res.Limit)
				assert.EqualValues(t, 2, res.Offset)

				expected := []int{3, 4, 5}
				for i, m := range res.Hospitals {
					verify(expected[i], m)
				}
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "hospital")

		F.Insert(db, model.Hospital{}, 0, 10, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
			r["Name"] = fmt.Sprintf("病院-%04d", i)
		})

		verify = func(id int, actual *model.Hospital) {
			assert.EqualValues(t, id, actual.Id)
			assert.EqualValues(t, fmt.Sprintf("hospital-%04d", id), actual.Uuid)
			assert.EqualValues(t, fmt.Sprintf("病院-%04d", id), actual.Name)
		}
	})
}

func TestAnnotationHospital_Fetch(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Hospital)

	httpTests := test.HttpTests{
		{
			Name:    "取得",
			Method:  http.MethodGet,
			Path:    "/annotation/hospitals/hospital-0003",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Hospital{}).(*model.Hospital)

				verify(3, res)
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodGet,
			Path:    "/annotation/hospitals/hospital-0000",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "hospital")

		F.Insert(db, model.Hospital{}, 0, 10, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
			r["Name"] = fmt.Sprintf("病院-%04d", i)
		})

		verify = func(id int, actual *model.Hospital) {
			assert.EqualValues(t, id, actual.Id)
			assert.EqualValues(t, fmt.Sprintf("hospital-%04d", id), actual.Uuid)
			assert.EqualValues(t, fmt.Sprintf("病院-%04d", id), actual.Name)
		}
	})
}