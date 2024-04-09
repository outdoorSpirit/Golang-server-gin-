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

	"gopkg.in/gorp.v2"
	"github.com/stretchr/testify/assert"

	//C "github.com/spiker/spiker-server/constant"
	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func prepareAlerts(t *testing.T, db *gorp.DbMap, beginTime time.Time) func(int, *model.ComputedEventEntity) {
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
	F.Insert(db, model.Measurement{}, 0, 4, func(i int, r F.Record) {
		r["Code"] = fmt.Sprintf("measurement-%04d", i)
		r["PatientId"] = F.If(i <= 3, patients[0].Id, patients[1].Id)
		r["TerminalId"] = F.If(i <= 3, terminals[0].Id, terminals[1].Id)
		r["FirstTime"] = time.Unix(0, 0)
		r["LastTime"] = time.Now()
	})

	// m - ce
	// 1 - [1-5] // 3番目が2に上書き
	// 2 - []
	// 3 - [6-10] // 4番目が3に上書き
	// 4 - [11-15] // 別病院
	ces := []struct{ mid int; risk int; fm int }{
		{1, 5, 10}, {1, 2, 30}, {1, 3, 50}, {1, 4, 70}, {1, 1, 90},
		{3, 1, 20}, {3, 4, 40}, {3, 2, 60}, {3, 5, 80}, {3, 3, 100},
		{4, 3, 10}, {4, 4, 30}, {4, 5, 50}, {4, 4, 70}, {4, 3, 90},
	}
	F.Insert(db, model.ComputedEvent{}, 0, 15, func(i int, r F.Record) {
		r["MeasurementId"] = ces[i-1].mid
		r["Risk"] = ces[i-1].risk
		r["IsHidden"] = false
		r["RangeFrom"] = beginTime.Add(time.Duration(ces[i-1].fm)*time.Minute)
		r["RangeUntil"] = beginTime.Add(time.Duration(ces[i-1].fm+5)*time.Minute)
		r["Parameters"] = model.JSON([]byte("{}"))
	})

	// ce - ae
	// 3  - [1,2]
	// 9  - [3]
	F.Insert(db, model.AnnotatedEvent{}, 0, 3, func(i int, r F.Record) {
		r["MeasurementId"] = F.If(i <= 2, 1, 3)
		r["ComputedEventId"] = F.If(i <= 2, 3, 9)
		r["Risk"] = F.If(i == 5, nil, i*100)
		diff := F.If(i <= 10, 10 - i, i - 11).(int)
		r["RangeFrom"] = beginTime.Add(time.Duration(diff)*time.Hour)
		r["RangeUntil"] = beginTime.Add(time.Duration(diff)*time.Hour + time.Duration(30)*time.Minute)
		r["CreatedAt"] = beginTime.Add(time.Duration(i)*time.Hour)
	})

	return func(id int, actual *model.ComputedEventEntity) {
		assert.EqualValues(t, id, actual.Id)

		assert.EqualValues(t, ces[id-1].risk, *actual.Risk)
		assert.EqualValues(t, false, actual.IsHidden)

		assert.EqualValues(t, actual.RangeFrom.Unix(), beginTime.Add(time.Duration(ces[id-1].fm)*time.Minute).Unix())
		assert.EqualValues(t, actual.RangeUntil.Unix(), beginTime.Add(time.Duration(ces[id-1].fm+5)*time.Minute).Unix())

		// アノテート済みのイベントは返されないはず。
		assert.EqualValues(t, 0, len(actual.Annotations))

		mm := actual.Measurement
		assert.NotNil(t, mm)

		if id <= 5 {
			assert.EqualValues(t, 1, mm.Id)
			assert.EqualValues(t, fmt.Sprintf("measurement-%04d", mm.Id), mm.Code)
		} else if id <= 10 {
			assert.EqualValues(t, 3, mm.Id)
			assert.EqualValues(t, fmt.Sprintf("measurement-%04d", mm.Id), mm.Code)
		} else {
			assert.FailNow(t, fmt.Sprintf("Unexpected event was obtained: %d", id))
		}
	}
}

func TestAnnotationAlert_Collect(t *testing.T) {
	auth := (&F.AnnotationFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	// 始点を指定しないと現在日時が基準となるため、未来のデータを入れる。
	beginTime := time.Now().Add(time.Hour).In(time.UTC)

	var verify func(int, *model.ComputedEventEntity)

	prepare := func(req *http.Request) {
		if _, e := db.Exec(`UPDATE computed_event SET is_suspended = false`); e != nil {
			assert.FailNow(t, e.Error())
		}
		if _, e := db.Exec(`UPDATE measurement SET last_time = $1`, time.Now()); e != nil {
			assert.FailNow(t, e.Error())
		}
	}

	httpTests := test.HttpTests{
		{
			Name:    "全未読イベント",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/alerts",
			Prepare: prepare,
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &collectUnreadAlertResponse{}).(*collectUnreadAlertResponse)

				assert.EqualValues(t, 4, len(res.Events))

				expected := []int{1, 7, 4, 10}
				for i, m := range res.Events {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "直近の計測が無いものは除く",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/alerts",
			Prepare: func(req *http.Request) {
				prepare(req)
				if _, e := db.Exec(
					`UPDATE measurement SET last_time = $1 WHERE id = 3`,
					time.Now().Add(time.Duration(-6)*time.Minute),
				); e != nil {
					assert.FailNow(t, e.Error())
				}
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &collectUnreadAlertResponse{}).(*collectUnreadAlertResponse)

				assert.EqualValues(t, 2, len(res.Events))

				expected := []int{1, 4}
				for i, m := range res.Events {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "待機中は除く",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/alerts",
			Prepare: func(req *http.Request) {
				prepare(req)
				if _, e := db.Exec(`UPDATE computed_event SET is_suspended = true WHERE id = 4`); e != nil {
					assert.FailNow(t, e.Error())
				}
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &collectUnreadAlertResponse{}).(*collectUnreadAlertResponse)

				assert.EqualValues(t, 3, len(res.Events))

				expected := []int{1, 7, 10}
				for i, m := range res.Events {
					verify(expected[i], m)
				}
			},
		},
		{
			Name:    "既読指定",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/annotation/hospitals/hospital-0001/alerts",
			Prepare: prepare,
			Query:   func(q url.Values) {
				q.Add("latest", "3")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &collectUnreadAlertResponse{}).(*collectUnreadAlertResponse)

				assert.EqualValues(t, 2, len(res.Events))

				expected := []int{4, 10}
				for i, m := range res.Events {
					verify(expected[i], m)
				}
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		verify = prepareAlerts(t, db, beginTime)
	})
}