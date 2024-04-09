package monitor

import (
	"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	//"strings"
	"testing"
	"time"

	"gopkg.in/gorp.v2"
	"github.com/stretchr/testify/assert"

	//C "github.com/spiker/spiker-server/constant"
	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func prepareEvents(t *testing.T, db *gorp.DbMap, beginTime time.Time) (func(int, *model.ComputedEventEntity), func(int, *model.AnnotatedEventEntity)) {
	F.Truncate(db, "annotated_event", "computed_event", "measurement_terminal", "patient", "measurement")

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
	measurements := F.Insert(db, model.Measurement{}, 0, 5, func(i int, r F.Record) {
		r["Code"] = fmt.Sprintf("measurement-%04d", i)
		r["PatientId"] = F.If(i <= 3, patients[0].Id, patients[1].Id)
		r["TerminalId"] = F.If(i <= 3, terminals[0].Id, terminals[1].Id)
	}).([]*model.Measurement)

	// m - ce
	// 1 - [1-10]
	// 2 - []
	// 3 - [11-15]
	F.Insert(db, model.ComputedEvent{}, 0, 15, func(i int, r F.Record) {
		r["MeasurementId"] = measurements[F.If(i <= 10, 0, 2).(int)].Id
		r["Risk"] = F.If(i == 5, nil, i*10)
		r["IsHidden"] = i == 3 || i == 8
		diff := F.If(i <= 10, 10 - i, i - 11).(int)
		r["RangeFrom"] = beginTime.Add(time.Duration(diff)*time.Hour)
		r["RangeUntil"] = beginTime.Add(time.Duration(diff)*time.Hour + time.Duration(30)*time.Minute)
		r["Parameters"] = model.JSON([]byte("{}"))
	})

	// m - ae
	// 1 - [1-10]
	// 2 - []
	// 3 - [11-15]
	// ce - ae
	// 1  - [1,3,5]
	// 3  - [2,4]
	F.Insert(db, model.AnnotatedEvent{}, 0, 15, func(i int, r F.Record) {
		r["MeasurementId"] = measurements[F.If(i <= 10, 0, 2).(int)].Id
		switch i {
		case 1, 3, 5:
			r["ComputedEventId"] = 1
		case 2, 4:
			r["ComputedEventId"] = 3
		default:
			r["ComputedEventId"] = nil
		}
		r["Risk"] = F.If(i == 5, nil, i*100)
		diff := F.If(i <= 10, 10 - i, i - 11).(int)
		r["RangeFrom"] = beginTime.Add(time.Duration(diff)*time.Hour)
		r["RangeUntil"] = beginTime.Add(time.Duration(diff)*time.Hour + time.Duration(30)*time.Minute)
		r["CreatedAt"] = beginTime.Add(time.Duration(i)*time.Hour)
		r["IsClosed"] = false
	})

	verifyComputed := func(id int, actual *model.ComputedEventEntity) {
		assert.EqualValues(t, id, actual.Id)

		if id == 5 {
			assert.Nil(t, actual.Risk)
		} else {
			assert.EqualValues(t, id*10, *actual.Risk)
		}

		assert.EqualValues(t, id == 3 || id == 8, actual.IsHidden)

		diff := F.If(id <= 10, 10 - id, id - 11).(int)
		assert.EqualValues(t, actual.RangeFrom, beginTime.Add(time.Duration(diff)*time.Hour))
		assert.EqualValues(t, actual.RangeUntil, beginTime.Add(time.Duration(diff)*time.Hour + time.Duration(30)*time.Minute))

		switch id {
		case 1:
			assert.EqualValues(t, 3, len(actual.Annotations))
			assert.EqualValues(t, 5, actual.Annotations[0].Id)
			assert.EqualValues(t, 3, actual.Annotations[1].Id)
			assert.EqualValues(t, 1, actual.Annotations[2].Id)
		case 3:
			assert.EqualValues(t, 2, len(actual.Annotations))
			assert.EqualValues(t, 4, actual.Annotations[0].Id)
			assert.EqualValues(t, 2, actual.Annotations[1].Id)
		default:
			assert.EqualValues(t, 0, len(actual.Annotations))
		}

		mm := actual.Measurement
		assert.NotNil(t, mm)

		if id <= 10 {
			assert.EqualValues(t, 1, mm.Id)
			assert.EqualValues(t, fmt.Sprintf("measurement-%04d", mm.Id), mm.Code)
		} else if id <= 15 {
			assert.EqualValues(t, 3, mm.Id)
			assert.EqualValues(t, fmt.Sprintf("measurement-%04d", mm.Id), mm.Code)
		}
	}

	verifyAnnotated := func(id int, actual *model.AnnotatedEventEntity) {
		assert.EqualValues(t, id, actual.Id)

		if id == 5 {
			assert.Nil(t, actual.Risk)
		} else {
			assert.EqualValues(t, id*100, *actual.Risk)
		}

		switch id {
		case 1, 3, 5:
			assert.NotNil(t, actual.Event)
			assert.EqualValues(t, 1, actual.Event.Id)
		case 2, 4:
			assert.NotNil(t, actual.Event)
			assert.EqualValues(t, 3, actual.Event.Id)
		default:
			assert.Nil(t, actual.Event)
		}

		diff := F.If(id <= 10, 10 - id, id - 11).(int)
		assert.EqualValues(t, actual.RangeFrom, beginTime.Add(time.Duration(diff)*time.Hour))
		assert.EqualValues(t, actual.RangeUntil, beginTime.Add(time.Duration(diff)*time.Hour + time.Duration(30)*time.Minute))
	}

	return verifyComputed, verifyAnnotated
}

func TestMonitorEvent_ListComputed(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.ComputedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "イベント一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/events",
			Query:   func(q url.Values) {
				q.Add("minutes", "300")
				q.Add("end", "2021-10-01T19:15:00+00:00")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listComputedEventsResponse{}).(*listComputedEventsResponse)

				assert.EqualValues(t, 4, len(res.Events))

				// hiddenは返らないのがアノテーション側との違い。
				expected := []int{4, 5, 6, 7}
				for i, m := range res.Events {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "結果が空",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/events",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listComputedEventsResponse{}).(*listComputedEventsResponse)

				assert.EqualValues(t, 0, len(res.Events))
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/events",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		verify, _ = prepareEvents(t, db, beginTime)
	})
}

func TestMonitorEvent_FetchComputed(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.ComputedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "イベント取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/events/1",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.ComputedEventEntity{}).(*model.ComputedEventEntity)

				verify(1, res)
			},
		},
		{
			Name:    "アノテーションなし",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/events/2",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.ComputedEventEntity{}).(*model.ComputedEventEntity)

				verify(2, res)
			},
		},
		{
			Name:    "イベントが無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/events/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/events/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "イベントの計測が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/events/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		verify, _ = prepareEvents(t, db, beginTime)
	})
}

func TestMonitorEvent_ListAnnotated(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.AnnotatedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "アノテーション一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/annotations",
			Query:   func(q url.Values) {
				q.Add("minutes", "300")
				q.Add("end", "2021-10-01T19:15:00+00:00")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listAnnotatedEventsResponse{}).(*listAnnotatedEventsResponse)

				assert.EqualValues(t, 6, len(res.Annotations))

				expected := []int{3, 4, 5, 6, 7, 8}
				for i, m := range res.Annotations {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "結果が空",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/annotations",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listAnnotatedEventsResponse{}).(*listAnnotatedEventsResponse)

				assert.EqualValues(t, 0, len(res.Annotations))
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/annotations",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		_, verify = prepareEvents(t, db, beginTime)
	})
}

func TestMonitorEvent_FetchAnnotated(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.AnnotatedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "アノテーション取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.AnnotatedEventEntity{}).(*model.AnnotatedEventEntity)

				verify(5, res)
			},
		},
		{
			Name:    "参照イベントが未設定",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/annotations/6",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.AnnotatedEventEntity{}).(*model.AnnotatedEventEntity)

				verify(6, res)
			},
		},
		{
			Name:    "アノテーションが無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/annotations/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "アノテーションの計測が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		_, verify = prepareEvents(t, db, beginTime)
	})
}

func TestMonitorEvent_CloseAnnotated(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	httpTests := test.HttpTests{
		{
			Name:    "イベントクローズ",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/annotations/5/close",
			Body:    test.JsonBody(map[string]interface{}{
				"memo": "メモ",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				if actual, e := db.Get(model.AnnotatedEvent{}, 5); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					ae := actual.(*model.AnnotatedEvent)
					assert.EqualValues(t, true, ae.IsClosed)
					assert.EqualValues(t, "メモ", *ae.ClosingMemo)
					assert.NotNil(t, ae.ClosedAt)
				}
			},
		},
		{
			Name:    "アノテーションが無い",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/1/measurements/1/annotations/0/close",
			Body:    test.JsonBody(map[string]interface{}{
				"memo": "メモ",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/annotations/5/close",
			Body:    test.JsonBody(map[string]interface{}{
				"memo": "メモ",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "アノテーションの計測が違う",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/annotations/5/close",
			Body:    test.JsonBody(map[string]interface{}{
				"memo": "メモ",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		_, _ = prepareEvents(t, db, beginTime)
	})
}