package monitor

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

func TestMonitorTerminal_List(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.MeasurementTerminal)

	httpTests := test.HttpTests{
		{
			Name:    "計測端末一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/terminals",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listTerminalsResponse{}).(*listTerminalsResponse)

				assert.EqualValues(t, 7, res.Total)
				assert.EqualValues(t, 100, res.Limit)
				assert.EqualValues(t, 0, res.Offset)

				assert.EqualValues(t, 7, len(res.Terminals))

				expected := []int{1, 2, 3, 4, 5, 7, 9}
				for i, m := range res.Terminals {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "範囲指定",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/terminals",
			Query:   func(q url.Values) {
				q.Add("limit", "3")
				q.Add("offset", "2")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listTerminalsResponse{}).(*listTerminalsResponse)

				assert.EqualValues(t, 7, res.Total)
				assert.EqualValues(t, 3, res.Limit)
				assert.EqualValues(t, 2, res.Offset)

				assert.EqualValues(t, 3, len(res.Terminals))

				expected := []int{3, 4, 5}
				for i, m := range res.Terminals {
					verify(expected[i], m)
				}
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement_terminal")

		// h - t
		// 1 - [1,2,3,4,5,7,9]
		// 2 - [6,8,10]
		// 3 - []
		F.Insert(db, model.MeasurementTerminal{}, 0, 10, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 5 || i%2 == 1, 1, 2)
			r["Code"] = fmt.Sprintf("terminal-%04d", i)
		})

		verify = func(id int, actual *model.MeasurementTerminal) {
			assert.EqualValues(t, id, actual.Id)

			// レスポンスに病院IDは含まれない。
			assert.EqualValues(t, 0, actual.HospitalId)
			assert.EqualValues(t, fmt.Sprintf("terminal-%04d", id), actual.Code)
		}
	})
}

func TestMonitorTerminal_Fetch(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verify func(int, *model.MeasurementTerminal)

	httpTests := test.HttpTests{
		{
			Name:    "患者取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/terminals/3",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.MeasurementTerminal{}).(*model.MeasurementTerminal)

				verify(3, res)
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/terminals/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/terminals/6",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement_terminal")

		// h - p
		// 1 - [1,2,3,4,5,7,9]
		// 2 - [6,8,10]
		// 3 - []
		F.Insert(db, model.MeasurementTerminal{}, 0, 10, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i <= 5 || i%2 == 1, 1, 2)
			r["Code"] = fmt.Sprintf("terminal-%04d", i)
		})

		verify = func(id int, actual *model.MeasurementTerminal) {
			assert.EqualValues(t, id, actual.Id)

			// レスポンスに病院IDは含まれない。
			assert.EqualValues(t, 0, actual.HospitalId)
			assert.EqualValues(t, fmt.Sprintf("terminal-%04d", id), actual.Code)
		}
	})
}

func TestMonitorTerminal_Update(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	testBody := func() map[string]interface{} {
		return map[string]interface{}{
			"memo": "メモ",
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "患者更新",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/terminals/1",
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.MeasurementTerminal{}).(*model.MeasurementTerminal)

				assert.EqualValues(t, 1, res.Id)

				var actual *model.MeasurementTerminal
				if r, e := db.Get(model.MeasurementTerminal{}, 1); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, r)
					actual = r.(*model.MeasurementTerminal)
				}

				verify := func(terminal *model.MeasurementTerminal) {
					assert.EqualValues(t, "メモ", terminal.Memo)
				}

				verify(res)
				verify(actual)
			},
		},
		{
			Name:    "存在しない",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/terminals/0",
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/1/terminals/2",
			Body:    test.JsonBody(testBody()),
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
			Path:    "/1/terminals/5",
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
		F.Truncate(db, "measurement_terminal")

		// h - p
		// 1 - [1,3,5]
		// 2 - [2,4]
		// 3 - []
		F.Insert(db, model.MeasurementTerminal{}, 0, 5, func(i int, r F.Record) {
			r["HospitalId"] = F.If(i%2 == 1, 1, 2)
			r["Code"] = fmt.Sprintf("terminal-%04d", i)
			r["Memo"] = fmt.Sprintf("memo-%04d", i)
		})
	})
}