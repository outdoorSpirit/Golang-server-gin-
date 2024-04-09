package admin

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

func TestAdminHospital_List(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Hospital)

	httpTests := test.HttpTests{
		{
			Name:    "一覧",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals",
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
			Path:    "/admin/hospitals",
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

func TestAdminHospital_Fetch(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Hospital)

	httpTests := test.HttpTests{
		{
			Name:    "取得",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/3",
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
			Path:    "/admin/hospitals/0",
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

func TestAdminHospital_Create(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	testBody := func()map[string]interface{} {
		return map[string]interface{}{
			"name": "新規病院",
			"memo": "新規メモ",
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "登録",
			Method:  http.MethodPost,
			Path:    "/admin/hospitals",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Hospital{}).(*model.Hospital)

				verify := func(m *model.Hospital) {
					assert.EqualValues(t, 11, m.Id)
					assert.NotEqual(t, "", m.Uuid)
					assert.NotEqual(t, "", m.Topic)
					assert.EqualValues(t, "新規病院", m.Name)
					assert.EqualValues(t, "新規メモ", m.Memo)
				}

				verify(res)

				if m, e := db.Get(model.Hospital{}, 11); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					verify(m.(*model.Hospital))
				}
			},
		},
	}

	validation := func(title string, key string, value interface{}) test.HttpTest {
		body := testBody()
		body[key] = value

		return test.HttpTest{
			Name:    title,
			Method:  http.MethodPost,
			Path:    "/admin/hospitals",
			Token:   auth.Token(1),
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("名前が空", "name", ""),
	)

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "hospital")

		F.Insert(db, model.Hospital{}, 0, 10, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
			r["Name"] = fmt.Sprintf("病院-%04d", i)
		})
	})
}

func TestAdminHospital_Update(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	testBody := func()map[string]interface{} {
		return map[string]interface{}{
			"name": "更新病院",
			"memo": "更新メモ",
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "更新",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/3",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Hospital{}).(*model.Hospital)

				verify := func(m *model.Hospital) {
					assert.EqualValues(t, 3, m.Id)
					assert.EqualValues(t, "更新病院", m.Name)
					assert.EqualValues(t, "更新メモ", m.Memo)
				}

				verify(res)

				if m, e := db.Get(model.Hospital{}, 3); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					verify(m.(*model.Hospital))
				}
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/0",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	validation := func(title string, key string, value interface{}) test.HttpTest {
		body := testBody()
		body[key] = value

		return test.HttpTest{
			Name:    title,
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/3",
			Token:   auth.Token(1),
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("名前が空", "name", ""),
	)

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "hospital")

		F.Insert(db, model.Hospital{}, 0, 10, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
			r["Name"] = fmt.Sprintf("病院-%04d", i)
		})
	})
}

func TestAdminHospital_Delete(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	httpTests := test.HttpTests{
		{
			Name:    "削除",
			Method:  http.MethodDelete,
			Path:    "/admin/hospitals/3",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				assert.EqualValues(t, 9, F.Count(t, db, "hospital", nil))

				if m, e := db.Get(model.Hospital{}, 3); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.Nil(t, m)
				}
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodDelete,
			Path:    "/admin/hospitals/0",
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
	})
}

func TestAdminHospital_GenerateApiKey(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	httpTests := test.HttpTests{
		{
			Name:    "作成",
			Method:  http.MethodPost,
			Path:    "/admin/hospitals/3/api_key",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.CTGAuthentication{}).(*model.CTGAuthentication)

				assert.NotEqual(t, int(0), res.Id)

				if m, e := db.Get(model.CTGAuthentication{}, res.Id); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					actual := m.(*model.CTGAuthentication)
					assert.EqualValues(t, 3, actual.HospitalId)
					assert.EqualValues(t, res.ApiKey, actual.ApiKey)
				}
			},
		},
		{
			Name:    "追加",
			Method:  http.MethodPost,
			Path:    "/admin/hospitals/1/api_key",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.CTGAuthentication{}).(*model.CTGAuthentication)

				assert.NotEqual(t, int(0), res.Id)

				if m, e := db.Get(model.CTGAuthentication{}, res.Id); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					actual := m.(*model.CTGAuthentication)
					assert.EqualValues(t, 1, actual.HospitalId)
					assert.EqualValues(t, res.ApiKey, actual.ApiKey)
				}
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPost,
			Path:    "/admin/hospitals/0/api_key",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "ctg_authentication", "hospital")

		F.Insert(db, model.Hospital{}, 0, 5, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
			r["Name"] = fmt.Sprintf("病院-%04d", i)
		})

		// 病院1に一つ、病院2に二つAPIキーを作成。
		F.Insert(db, model.CTGAuthentication{}, 0, 3, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i==3, 2, 1)
			r["ApiKey"] = fmt.Sprintf("api_key-%04d", i)
		})
	})
}