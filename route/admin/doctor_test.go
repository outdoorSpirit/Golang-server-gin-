package admin

import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	//"time"

	"github.com/stretchr/testify/assert"

	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestAdminDoctor_List(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Doctor, bool)

	httpTests := test.HttpTests{
		{
			Name:    "一覧",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/3/doctors",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listDoctorsResponse{}).(*listDoctorsResponse)

				assert.EqualValues(t, 10, len(res.Doctors))
				assert.EqualValues(t, 10, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				expected := []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
				for i, m := range res.Doctors {
					verify(expected[i], m, false)
				}
			},
		},
		{
			Name:    "空",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/2/doctors",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listDoctorsResponse{}).(*listDoctorsResponse)

				assert.EqualValues(t, 0, len(res.Doctors))
				assert.EqualValues(t, 0, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)
			},
		},
		{
			Name:    "範囲指定",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/3/doctors",
			Token:   auth.Token(1),
			Query:   func(q url.Values) {
				q.Add("limit", "3")
				q.Add("offset", "2")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listDoctorsResponse{}).(*listDoctorsResponse)

				assert.EqualValues(t, 3, len(res.Doctors))
				assert.EqualValues(t, 10, res.Total)
				assert.EqualValues(t, 3, res.Limit)
				assert.EqualValues(t, 2, res.Offset)

				expected := []int{6, 8, 10}
				for i, m := range res.Doctors {
					verify(expected[i], m, false)
				}
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/0/doctors",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				// REVIEW 病院が無い場合も、空の200レスポンスとなる。
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listDoctorsResponse{}).(*listDoctorsResponse)

				assert.EqualValues(t, 0, len(res.Doctors))
				assert.EqualValues(t, 0, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "doctor", "hospital")

		// h - d
		// 1 - [1,3,5,...,19]
		// 2 - []
		// 3 - [2,4,6,...,20]
		F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		})

		F.Insert(db, model.Doctor{}, 0, 20, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 3)
			r["LoginId"] = fmt.Sprintf("doctor-%04d", i)
			r["Name"] = fmt.Sprintf("医師-%04d", i)
		})

		verify = func(id int, actual *model.Doctor, fromDB bool) {
			assert.EqualValues(t, id, actual.Id)
			assert.EqualValues(t, fmt.Sprintf("医師-%04d", id), actual.Name)

			if fromDB {
				if id % 2 == 1 {
					assert.EqualValues(t, 1, actual.HospitalId)
				} else {
					assert.EqualValues(t, 3, actual.HospitalId)
				}

				assert.NotEqual(t, "", actual.Password)
				assert.NotEqual(t, "", actual.TokenVersion)
			} else {
				assert.EqualValues(t, 0, actual.HospitalId)
				assert.EqualValues(t, "", actual.Password)
				assert.EqualValues(t, "", actual.TokenVersion)
			}
		}
	})
}

func TestAdminDoctor_Fetch(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.Doctor, bool)

	httpTests := test.HttpTests{
		{
			Name:    "取得",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/3/doctors/8",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Doctor{}).(*model.Doctor)

				verify(8, res, false)
			},
		},
		{
			Name:    "医者と病院の組み合わせが正しくない",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/1/doctors/8",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/0/doctors/8",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "医者が無い",
			Method:  http.MethodGet,
			Path:    "/admin/hospitals/3/doctors/0",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "doctor", "hospital")

		// h - d
		// 1 - [1,3,5,...,19]
		// 2 - []
		// 3 - [2,4,6,...,20]
		F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		})

		F.Insert(db, model.Doctor{}, 0, 20, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 3)
			r["LoginId"] = fmt.Sprintf("doctor-%04d", i)
			r["Name"] = fmt.Sprintf("医師-%04d", i)
		})

		verify = func(id int, actual *model.Doctor, fromDB bool) {
			assert.EqualValues(t, id, actual.Id)
			assert.EqualValues(t, fmt.Sprintf("医師-%04d", id), actual.Name)

			if fromDB {
				if id % 2 == 1 {
					assert.EqualValues(t, 1, actual.HospitalId)
				} else {
					assert.EqualValues(t, 3, actual.HospitalId)
				}

				assert.NotEqual(t, "", actual.Password)
				assert.NotEqual(t, "", actual.TokenVersion)
			} else {
				assert.EqualValues(t, 0, actual.HospitalId)
				assert.EqualValues(t, "", actual.Password)
				assert.EqualValues(t, "", actual.TokenVersion)
			}
		}
	})
}

func TestAdminDoctor_Create(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	testBody := func() map[string]interface{}{
		return map[string]interface{}{
			"loginId": "new-doctor",
			"password": "new-password",
			"name": "新規医者",
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "登録",
			Method:  http.MethodPost,
			Path:    "/admin/hospitals/3/doctors",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Doctor{}).(*model.Doctor)

				assert.EqualValues(t, 21, F.Count(t, db, "doctor", nil))

				verify := func(m *model.Doctor, fromDB bool) {
					assert.EqualValues(t, 21, m.Id)
					assert.EqualValues(t, "新規医者", m.Name)

					if fromDB {
						assert.EqualValues(t, 3, m.HospitalId)
						assert.NotEqual(t, "", m.TokenVersion)

						if p, e := db.SelectStr(`SELECT encode(digest($1, 'sha256'), 'hex')`, "new-password"); e != nil {
							assert.FailNow(t, e.Error())
						} else {
							assert.EqualValues(t, p, m.Password)
						}
					} else {
						assert.EqualValues(t, 0, m.HospitalId)
						assert.EqualValues(t, "", m.Password)
						assert.EqualValues(t, "", m.TokenVersion)
					}
				}

				verify(res, false)

				if m, e := db.Get(model.Doctor{}, 21); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, m)
					verify(m.(*model.Doctor), true)
				}
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPost,
			Path:    "/admin/hospitals/0/doctors",
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
			Method:  http.MethodPost,
			Path:    "/admin/hospitals/3/doctors",
			Token:   auth.Token(1),
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("ログインIDが空", "loginId", ""),
		validation("ログインIDが長い", "loginId", strings.Repeat("abc", 11)),
		validation("ログインID重複", "loginId", "doctor-0003"),
		validation("パスワードが空", "password", ""),
		validation("パスワードが長い", "password", strings.Repeat("abc", 11)),
		validation("名前が空", "name", ""),
		validation("名前が長い", "name", strings.Repeat("abcde", 13)),
	)

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "doctor", "hospital")

		// h - d
		// 1 - [1,3,5,...,19]
		// 2 - []
		// 3 - [2,4,6,...,20]
		F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		})

		F.Insert(db, model.Doctor{}, 0, 20, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 3)
			r["LoginId"] = fmt.Sprintf("doctor-%04d", i)
			r["Name"] = fmt.Sprintf("医師-%04d", i)
		})
	})
}

func TestAdminDoctor_UpdateProfile(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	testBody := func() map[string]interface{}{
		return map[string]interface{}{
			"name": "更新医者",
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "更新",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/3/doctors/8",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.Doctor{}).(*model.Doctor)

				verify := func(m *model.Doctor, fromDB bool) {
					assert.EqualValues(t, 8, m.Id)
					assert.EqualValues(t, "更新医者", m.Name)
				}

				verify(res, false)

				if m, e := db.Get(model.Doctor{}, 8); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, m)
					verify(m.(*model.Doctor), true)
				}
			},
		},
		{
			Name:    "医者と病院の組み合わせが正しくない",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/1/doctors/8",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/0/doctors/8",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "医者が無い",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/3/doctors/0",
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
			Path:    "/admin/hospitals/3/doctors/8",
			Token:   auth.Token(1),
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("名前が空", "name", ""),
		validation("名前が長い", "name", strings.Repeat("abcde", 13)),
	)

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "doctor", "hospital")

		// h - d
		// 1 - [1,3,5,...,19]
		// 2 - []
		// 3 - [2,4,6,...,20]
		F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		})

		F.Insert(db, model.Doctor{}, 0, 20, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 3)
			r["LoginId"] = fmt.Sprintf("doctor-%04d", i)
			r["Name"] = fmt.Sprintf("医師-%04d", i)
		})
	})
}

func TestAdminDoctor_UpdatePassword(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	testBody := func() map[string]interface{}{
		return map[string]interface{}{
			"password": "update-password",
		}
	}

	var passwordHash string

	if p, e := db.SelectStr(`SELECT encode(digest($1, 'sha256'), 'hex')`, "update-password"); e != nil {
		assert.FailNow(t, e.Error())
	} else {
		passwordHash = p
	}

	httpTests := test.HttpTests{
		{
			Name:    "更新",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/3/doctors/8/password",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				verify := func(m *model.Doctor, fromDB bool) {
					assert.EqualValues(t, 8, m.Id)
					assert.EqualValues(t, passwordHash, m.Password)
				}

				if m, e := db.Get(model.Doctor{}, 8); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, m)
					verify(m.(*model.Doctor), true)
				}
			},
		},
		{
			Name:    "医者と病院の組み合わせが正しくない",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/1/doctors/8/password",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/0/doctors/8/password",
			Token:   auth.Token(1),
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "医者が無い",
			Method:  http.MethodPut,
			Path:    "/admin/hospitals/3/doctors/0/password",
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
			Path:    "/admin/hospitals/3/doctors/8/password",
			Token:   auth.Token(1),
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("パスワードが空", "password", ""),
		validation("パスワードが長い", "password", strings.Repeat("abc", 11)),
	)

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "doctor", "hospital")

		// h - d
		// 1 - [1,3,5,...,19]
		// 2 - []
		// 3 - [2,4,6,...,20]
		F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		})

		F.Insert(db, model.Doctor{}, 0, 20, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 3)
			r["LoginId"] = fmt.Sprintf("doctor-%04d", i)
			r["Name"] = fmt.Sprintf("医師-%04d", i)
		})
	})
}

func TestAdminDoctor_Delete(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	httpTests := test.HttpTests{
		{
			Name:    "削除",
			Method:  http.MethodDelete,
			Path:    "/admin/hospitals/3/doctors/8",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				assert.EqualValues(t, 19, F.Count(t, db, "doctor", nil))

				if m, e := db.Get(model.Doctor{}, 8); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.Nil(t, m)
				}
			},
		},
		{
			Name:    "医者と病院の組み合わせが正しくない",
			Method:  http.MethodDelete,
			Path:    "/admin/hospitals/1/doctors/6",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodDelete,
			Path:    "/admin/hospitals/0/doctors/6",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "医者が無い",
			Method:  http.MethodDelete,
			Path:    "/admin/hospitals/3/doctors/0",
			Token:   auth.Token(1),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "doctor", "hospital")

		// h - d
		// 1 - [1,3,5,...,19]
		// 2 - []
		// 3 - [2,4,6,...,20]
		F.Insert(db, model.Hospital{}, 0, 3, func(i int, r F.Record) {
			r["Uuid"] = fmt.Sprintf("hospital-%04d", i)
		})

		F.Insert(db, model.Doctor{}, 0, 20, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 3)
			r["LoginId"] = fmt.Sprintf("doctor-%04d", i)
			r["Name"] = fmt.Sprintf("医師-%04d", i)
		})
	})
}