package annotation

import (
	"fmt"
	//"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	F.Truncate(db, "annotated_event", "computed_event", "measurement_terminal", "patient", "measurement", "hospital")

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
		r["Parameters"] = model.JSON([]byte(fmt.Sprintf(`{"value":%d}`, i)))
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

func TestAnnotationEvent_ListComputed(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.ComputedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "イベント一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events",
			Query:   func(q url.Values) {
				q.Add("minutes", "300")
				q.Add("end", "2021-10-01T19:15:00+00:00")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listComputedEventsResponse{}).(*listComputedEventsResponse)

				assert.EqualValues(t, 6, len(res.Events))

				expected := []int{3, 4, 5, 6, 7, 8}
				for i, m := range res.Events {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "結果が空",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/events",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listComputedEventsResponse{}).(*listComputedEventsResponse)

				assert.EqualValues(t, 0, len(res.Events))
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/2/events",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/events",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		verify, _ = prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_FetchComputed(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.ComputedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "イベント取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/1",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/2",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/1/events/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/events/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "イベントの計測が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/2/events/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		verify, _ = prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_ShowComputed(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	prepare := func(req *http.Request) {
		if _, e := db.Exec(`UPDATE computed_event SET is_hidden = true WHERE id = 3`); e != nil {
			assert.FailNow(t, e.Error())
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "更新",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/3/show",
			Prepare: prepare,
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				if actual, e := db.Get(model.ComputedEvent{}, 3); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					evt := actual.(*model.ComputedEvent)
					assert.EqualValues(t, false, evt.IsHidden)
				}

				if actual, e := db.Get(model.ComputedEvent{}, 8); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					evt := actual.(*model.ComputedEvent)
					assert.EqualValues(t, true, evt.IsHidden)
				}
			},
		},
		{
			Name:    "イベントが無い",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/0/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/1/events/3/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/events/3/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "イベントの計測が違う",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/2/events/3/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_HideComputed(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	prepare := func(req *http.Request) {
		if _, e := db.Exec(`UPDATE computed_event SET is_hidden = false WHERE id = 5`); e != nil {
			assert.FailNow(t, e.Error())
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "更新",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/5/show",
			Prepare: prepare,
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				if actual, e := db.Get(model.ComputedEvent{}, 5); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					evt := actual.(*model.ComputedEvent)
					assert.EqualValues(t, true, evt.IsHidden)
				}

				if actual, e := db.Get(model.ComputedEvent{}, 4); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					evt := actual.(*model.ComputedEvent)
					assert.EqualValues(t, false, evt.IsHidden)
				}
			},
		},
		{
			Name:    "イベントが無い",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/0/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/1/events/5/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/events/5/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "イベントの計測が違う",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/2/events/5/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_SuspendComputed(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	prepare := func(req *http.Request) {
		if _, e := db.Exec(`UPDATE computed_event SET is_suspended = false`); e != nil {
			assert.FailNow(t, e.Error())
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "待機",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/5/suspend",
			Prepare: prepare,
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				if actual, e := db.Get(model.ComputedEvent{}, 5); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					evt := actual.(*model.ComputedEvent)
					assert.True(t, evt.IsSuspended)
				}

				others := []*model.ComputedEvent{}
				if _, e := db.Select(&others, `SELECT * FROM computed_event WHERE id != 5`); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					for _, m := range others {
						assert.False(t, m.IsSuspended)
					}
				}
			},
		},
		{
			Name:    "イベントが無い",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/events/0/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/1/events/5/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/events/5/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "イベントの計測が違う",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/2/events/5/show",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_ListAnnotated(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.AnnotatedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "アノテーション一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/annotations",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listAnnotatedEventsResponse{}).(*listAnnotatedEventsResponse)

				assert.EqualValues(t, 0, len(res.Annotations))
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/2/annotations",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/annotations",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		_, verify = prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_FetchAnnotated(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	var verify func(int, *model.AnnotatedEventEntity)

	httpTests := test.HttpTests{
		{
			Name:    "アノテーション取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/5",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/6",
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
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/0",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/1/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "アノテーションの計測が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/2/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		_, verify = prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_RegisterAnnotated(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	verify := func(m *model.AnnotatedEvent, fromDB bool, id int, mid int, ceid int, risk int) {
		assert.EqualValues(t, id, m.Id)
		if fromDB {
			assert.EqualValues(t, mid, m.MeasurementId)
			assert.EqualValues(t, 1, *m.AnnotatorId)
			if ceid == 0 {
				assert.Nil(t, m.ComputedEventId)
			} else {
				assert.EqualValues(t, ceid, *m.ComputedEventId)
			}
		}
		assert.EqualValues(t, risk, *m.Risk)
		assert.EqualValues(t, "メモ", m.Memo)
		assert.EqualValues(t, time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC), m.RangeFrom)
		assert.EqualValues(t, time.Date(2021, time.October, 1, 13, 0, 0, 0, time.UTC), m.RangeUntil)
	}

	testBody := func(ceid int)map[string]interface{}{
		body := map[string]interface{}{
			"risk": 3,
			"memo": "メモ",
			"rangeFrom": "2021-10-01T12:00:00+00:00",
			"rangeUntil": "2021-10-01T13:00:00+00:00",
		}

		if ceid >= 0 {
			body["eventId"] = ceid
		}

		return body
	}

	prepare := func(req *http.Request) {
		// m - ma
		// 2 - 1(now)
		// 3 - 2(not now)
		now := time.Now()

		F.Truncate(db, "measurement_alert")
		F.Insert(db, model.MeasurementAlert{}, 0, 1, func(i int, r F.Record) {
			r["MeasurementId"] = i+1
			if i == 1 {
				r["SilentFrom"] = now.Add(time.Duration(-10)*time.Minute)
				r["SilentUntil"] = now.Add(time.Duration(10)*time.Minute)
			} else {
				r["SilentFrom"] = now.Add(time.Duration(-10)*time.Minute)
				r["SilentUntil"] = now.Add(time.Duration(-5)*time.Minute)
			}
		})
	}

	httpTests := test.HttpTests{
		{
			Name:    "アノテーション登録",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/annotations",
			Prepare: prepare,
			Body:    test.JsonBody(testBody(-1)),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.AnnotatedEvent{}).(*model.AnnotatedEvent)

				verify(res, false, 16, 2, 0, 3)

				if actual, e := db.Get(model.AnnotatedEvent{}, 16); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, actual)
					verify(actual.(*model.AnnotatedEvent), true, 16, 2, 0, 3)
				}

				alerts := []*model.MeasurementAlert{}
				if _, e := db.Select(&alerts, `SELECT * FROM measurement_alert WHERE measurement_id = 2`); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.EqualValues(t, 1, len(alerts))
					assert.True(t, alerts[0].SilentUntil.Before(time.Now()))
				}
			},
		},
		{
			Name:    "参照イベントを指定して登録",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/3/annotations",
			Prepare: prepare,
			Body:    test.JsonBody(testBody(13)),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.AnnotatedEvent{}).(*model.AnnotatedEvent)

				verify(res, false, 17, 3, 13, 3)

				if actual, e := db.Get(model.AnnotatedEvent{}, 17); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, actual)
					verify(actual.(*model.AnnotatedEvent), true, 17, 3, 13, 3)
				}
			},
		},
		{
			Name:    "リスクが低いためサイレント解除されない",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/annotations",
			Prepare: prepare,
			Body:    func()(io.Reader, string) {
				body := testBody(-1)
				body["risk"] = 2
				return test.JsonBody(body)()
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, rec.Code)
				res := F.FromJsonResponse(t, rec, &model.AnnotatedEvent{}).(*model.AnnotatedEvent)

				verify(res, false, 18, 2, 0, 2)

				if actual, e := db.Get(model.AnnotatedEvent{}, 18); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, actual)
					verify(actual.(*model.AnnotatedEvent), true, 18, 2, 0, 2)
				}

				alerts := []*model.MeasurementAlert{}
				if _, e := db.Select(&alerts, `SELECT * FROM measurement_alert WHERE measurement_id = 2`); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.EqualValues(t, 1, len(alerts))
					assert.True(t, alerts[0].SilentUntil.After(time.Now()))
				}
			},
		},
		{
			Name:    "参照イベントが計測に含まれない",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/annotations",
			Body:    test.JsonBody(testBody(13)),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			Name:    "参照イベントがない",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/annotations",
			Body:    test.JsonBody(testBody(0)),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			Name:    "計測が無い",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/0/annotations",
			Body:    test.JsonBody(testBody(-1)),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/2/annotations",
			Body:    test.JsonBody(testBody(-1)),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/2/annotations",
			Body:    test.JsonBody(testBody(-1)),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	validation := func(title string, key string, value interface{}) test.HttpTest {
		body := testBody(-1)
		body[key] = value

		return test.HttpTest{
			Name:    title,
			Method:  http.MethodPost,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/2/annotations",
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("リスクが無効", "risk", 6),
		validation("メモが長い", "memo", strings.Repeat("0123456789", 200) + "0"),
		validation("先頭日時が無効", "rangeFrom", "2020/10/1"),
		validation("末尾日時が無効", "rangeUntil", "2020/10/1"),
	)

	httpTests.Run(testHandler(), t, func() {
		prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_UpdateAnnotated(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	verify := func(m *model.AnnotatedEvent, fromDB bool, id int, mid int, risk int) {
		assert.EqualValues(t, id, m.Id)
		if fromDB {
			assert.EqualValues(t, mid, m.MeasurementId)
			assert.EqualValues(t, 1, *m.AnnotatorId)
		}
		assert.EqualValues(t, risk, *m.Risk)
		assert.EqualValues(t, "メモ", m.Memo)
	}

	testBody := func()map[string]interface{}{
		return map[string]interface{}{
			"risk": 3,
			"memo": "メモ",
		}
	}

	prepare := func(req *http.Request) {
		// m - ma
		// 1 - 1(now)
		// 2 - 2(not now)
		now := time.Now()

		F.Truncate(db, "measurement_alert")
		F.Insert(db, model.MeasurementAlert{}, 0, 1, func(i int, r F.Record) {
			r["MeasurementId"] = i
			if i == 1 {
				r["SilentFrom"] = now.Add(time.Duration(-10)*time.Minute)
				r["SilentUntil"] = now.Add(time.Duration(10)*time.Minute)
			} else {
				r["SilentFrom"] = now.Add(time.Duration(-10)*time.Minute)
				r["SilentUntil"] = now.Add(time.Duration(-5)*time.Minute)
			}
		})
	}

	httpTests := test.HttpTests{
		{
			Name:    "アノテーション更新",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/5",
			Prepare: prepare,
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				if actual, e := db.Get(model.AnnotatedEvent{}, 5); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, actual)
					verify(actual.(*model.AnnotatedEvent), true, 5, 1, 3)
				}

				alerts := []*model.MeasurementAlert{}
				if _, e := db.Select(&alerts, `SELECT * FROM measurement_alert WHERE measurement_id = 1`); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.EqualValues(t, 1, len(alerts))
					assert.True(t, alerts[0].SilentUntil.Before(time.Now()))
				}
			},
		},
		{
			Name:    "リスクが低いためサイレント解除されない",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/5",
			Prepare: prepare,
			Body:    func()(io.Reader, string) {
				body := testBody()
				body["risk"] = 2
				return test.JsonBody(body)()
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				if actual, e := db.Get(model.AnnotatedEvent{}, 5); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, actual)
					verify(actual.(*model.AnnotatedEvent), true, 5, 1, 2)
				}

				alerts := []*model.MeasurementAlert{}
				if _, e := db.Select(&alerts, `SELECT * FROM measurement_alert WHERE measurement_id = 1`); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.EqualValues(t, 1, len(alerts))
					assert.True(t, alerts[0].SilentUntil.After(time.Now()))
				}
			},
		},
		{
			Name:    "アノテーションが無い",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/0",
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測が無い",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/0/annotations/5",
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/1/annotations/5",
			Body:    test.JsonBody(testBody()),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodPut,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/annotations/5",
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
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/5",
			Body:    test.JsonBody(body),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}
	}

	httpTests = append(httpTests,
		validation("リスクが無効", "risk", 6),
		validation("メモが長い", "memo", strings.Repeat("0123456789", 200) + "0"),
	)

	httpTests.Run(testHandler(), t, func() {
		prepareEvents(t, db, beginTime)
	})
}

func TestAnnotationEvent_DeleteAnnotated(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	httpTests := test.HttpTests{
		{
			Name:    "アノテーション削除",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/1/annotations/4",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				if actual, e := db.Get(model.AnnotatedEvent{}, 4); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.Nil(t, actual)
				}
			},
		},
		{
			Name:    "計測が無い",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/measurements/0/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が無い",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0000/measurements/1/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "計測の病院が違う",
			Method:  http.MethodDelete,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0002/measurements/1/annotations/5",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		prepareEvents(t, db, beginTime)
	})
}