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

func prepareAlerts(t *testing.T, db *gorp.DbMap, beginTime time.Time) func(int, *model.AnnotatedEventEntity) {
	F.Truncate(db, "annotated_event", "computed_event", "measurement_terminal", "patient", "measurement")

	patients := F.Insert(db, model.Patient{}, 0, 3, func(i int, r F.Record) {
		r["HospitalId"] = i
	}).([]*model.Patient)

	terminals := F.Insert(db, model.MeasurementTerminal{}, 0, 3, func(i int, r F.Record) {
		r["HospitalId"] = i
	}).([]*model.MeasurementTerminal)

	// h - m
	// 1 - [1-10]
	// 2 - [11-15]
	// 3 - []
	F.Insert(db, model.Measurement{}, 0, 15, func(i int, r F.Record) {
		r["Code"] = fmt.Sprintf("measurement-%04d", i)
		r["PatientId"] = F.If(i <= 10, patients[0].Id, patients[1].Id)
		r["TerminalId"] = F.If(i <= 10, terminals[0].Id, terminals[1].Id)
	})

	// beginTimeから20分おきに、5分間の自動計測。
	// リスク値は影響しないため、全て5とする。
	// 病院が同じならば計測を問わないため、1と2に分ける。
	// m - ce
	// 1 - [1,3,5,7,9]
	// 2 - [2,4,6,8,10]
	// 3-7 - [11-15]
	F.Insert(db, model.ComputedEvent{}, 0, 15, func(i int, r F.Record) {
		r["MeasurementId"] = F.If(i <= 10, F.If(i%2 == 1, 1, 2), i-8)
		r["Risk"] = 5
		r["IsHidden"] = false
		diff := F.If(i <= 10, i-1, i-11).(int)
		r["RangeFrom"] = beginTime.Add(time.Duration(20*diff)*time.Minute)
		r["RangeUntil"] = beginTime.Add(time.Duration(20*diff+5)*time.Minute)
		r["Parameters"] = model.JSON([]byte("{}"))
	})

	// beginTimeから10分おきに5分間のアノテーションを以下のタイミングでつける。
	// 後の方が新しいアノテーション。
	// ce - ae
	// 1  - [1(1), 2(2), 3(5)]
	// x
	// 2  - [4(nil), 5(3)]
	// x  - [6(nil)]
	// 3
	// x  - [7(3)]
	// 4  - [8(4)]
	// x 
	// 5  - [9(1)]
	// x
	// 6  - [10(nil)]
	// x
	// 7
	// x  - [11(1)]
	// 8  - [12(2), 13(3)]
	// x
	// 9  - [14(4)]
	// x
	// 10 - [15(1)]
	c2a := []struct{ diff int; ceid int; risk int; }{
		{0, 1, 1}, {0, 1, 2}, {0, 1, 5},
		{2, 2, 0}, {2, 2, 3},
		{3, 0, 0},
		{5, 0, 3},
		{6, 4, 4},
		{8, 5, 1},
		{10, 6, 0},
		{13, 0, 1},
		{14, 8, 2}, {14, 8, 3},
		{16, 9, 4},
		{18, 10, 1},
	}
	F.Insert(db, model.AnnotatedEvent{}, 0, 15, func(i int, r F.Record) {
		ceid := c2a[i-1].ceid

		if ceid == 0 {
			r["MeasurementId"] = 1
			r["ComputedEventId"] = nil
		} else if ceid%2 == 1 {
			r["MeasurementId"] = 1
			r["ComputedEventId"] = &ceid
		} else {
			r["MeasurementId"] = 2
			r["ComputedEventId"] = &ceid
		}
		r["AnnotatorId"] = nil
		if c2a[i-1].risk != 0 {
			r["Risk"] = c2a[i-1].risk
		} else {
			r["Risk"] = nil
		}
		r["RangeFrom"] = beginTime.Add(time.Duration(c2a[i-1].diff*10)*time.Minute)
		r["RangeUntil"] = beginTime.Add(time.Duration(c2a[i-1].diff*10+5)*time.Minute)
		r["CreatedAt"] = beginTime.Add(time.Duration(i)*time.Minute)
	})

	// その他のアノテーション
	// m - a
	// 3 - [18,16,17]
	// 4 - []
	// 5 - [19]
	// 6 - [21,20]
	// 7 - []
	m2a := []struct{ mid int; risk int; diff int }{
		{3, 1, 2}, {3, 2, 3}, {3, 2, 1},
		{5, 0, 1},
		{6, 2, 2}, {6, 2, 1},
	}
	F.Insert(db, model.AnnotatedEvent{}, 0, 6, func(i int, r F.Record) {
		r["MeasurementId"] = m2a[i-1].mid
		if m2a[i-1].risk != 0 {
			r["Risk"] = m2a[i-1].risk
		} else {
			r["Risk"] = nil
		}
		r["RangeFrom"] = beginTime.Add(time.Duration(10*m2a[i-1].diff)*time.Minute)
		r["RangeUntil"] = beginTime.Add(time.Duration(10*m2a[i-1].diff+5)*time.Minute)
		r["CreatedAt"] = beginTime.Add(time.Duration(i)*time.Minute)
	})

	// 別病院
	F.Insert(db, model.AnnotatedEvent{}, 0, 5, func(i int, r F.Record) {
		r["MeasurementId"] = 11
		r["RangeFrom"] = beginTime.Add(time.Duration((i-1)*30+5)*time.Minute)
		r["RangeUntil"] = beginTime.Add(time.Duration((i-1)*30+10)*time.Minute)
		r["CreatedAt"] = beginTime.Add(time.Duration(i)*time.Minute)
	})

	verifyAnnotated := func(id int, actual *model.AnnotatedEventEntity) {
		assert.EqualValues(t, id, actual.Id)

		switch id {
		case 19:
			assert.Nil(t, actual.Risk)
		case 15, 16:
			assert.EqualValues(t, 1, *actual.Risk)
		case 17, 18, 20, 21:
			assert.EqualValues(t, 2, *actual.Risk)
		case 5, 7, 13:
			assert.EqualValues(t, 3, *actual.Risk)
		case 3:
			assert.EqualValues(t, 5, *actual.Risk)
		case 8, 14:
			assert.EqualValues(t, 4, *actual.Risk)
		default:
			assert.FailNow(t, fmt.Sprintf("Unexpeccted annotation #%d", id))
		}

		if id <= 15 {
			ceid := c2a[id-1].ceid
			if ceid == 0 {
				assert.Nil(t, actual.Event)
			} else {
				assert.EqualValues(t, ceid, actual.Event.Id)
			}

			assert.EqualValues(t, actual.RangeFrom.Unix(), beginTime.Add(time.Duration(c2a[id-1].diff*10)*time.Minute).Unix())
			assert.EqualValues(t, actual.RangeUntil.Unix(), beginTime.Add(time.Duration(c2a[id-1].diff*10+5)*time.Minute).Unix())

			mm := actual.Measurement
			assert.NotNil(t, mm)

			assert.EqualValues(t, F.If(ceid == 0, 1, F.If(ceid%2 == 1, 1, 2)), mm.Id)
			assert.EqualValues(t, fmt.Sprintf("measurement-%04d", mm.Id), mm.Code)
		} else if id <= 21 {
			assert.Nil(t, actual.Event)

			assert.EqualValues(t, actual.RangeFrom.Unix(), beginTime.Add(time.Duration(m2a[id-16].diff*10)*time.Minute).Unix())
			assert.EqualValues(t, actual.RangeUntil.Unix(), beginTime.Add(time.Duration(m2a[id-16].diff*10+5)*time.Minute).Unix())

			mm := actual.Measurement
			assert.NotNil(t, mm)

			assert.EqualValues(t, F.If(id <= 18, 3, F.If(id == 19, 5, 6)), mm.Id)
			assert.EqualValues(t, fmt.Sprintf("measurement-%04d", mm.Id), mm.Code)
		}
	}

	return verifyAnnotated
}

func TestMonitorAlert_Collect(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	var verifyAnnotated func(int, *model.AnnotatedEventEntity)

	beginTime := time.Date(2021, time.October, 1, 12, 0, 0, 0, time.UTC)

	httpTests := test.HttpTests{
		{
			Name:    "アラート一覧",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/alerts",
			Query:   func(q url.Values) {
				q.Add("measurements", "1")
				q.Add("measurements", "2")
				q.Add("measurements", "3")
				q.Add("measurements", "4")
				q.Add("measurements", "5")
			},
			Prepare: func(req *http.Request) {
				F.Truncate(db, "measurement_alert")

				_, err := db.Exec(`UPDATE annotated_event SET is_closed = false`)
				assert.NoError(t, err)
				_, err = db.Exec(`UPDATE measurement SET is_closed = false`)
				assert.NoError(t, err)
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &collectAlertsResponse{}).(*collectAlertsResponse)

				assert.EqualValues(t, 4, len(res.Annotations))

				expected := []int{19, 17, 14, 15}
				for i, m := range res.Annotations {
					verifyAnnotated(expected[i], m)
				}
			},
		},
		{
			Name:    "完了済みは返されない",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/alerts",
			Query:   func(q url.Values) {
				q.Add("measurements", "1")
				q.Add("measurements", "2")
				q.Add("measurements", "3")
				q.Add("measurements", "4")
				q.Add("measurements", "5")
			},
			Prepare: func(req *http.Request) {
				F.Truncate(db, "measurement_alert")

				_, err := db.Exec(`UPDATE annotated_event SET is_closed = true WHERE id = 17`)
				assert.NoError(t, err)
				_, err = db.Exec(`UPDATE measurement SET is_closed = true WHERE id = 1`)
				assert.NoError(t, err)
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &collectAlertsResponse{}).(*collectAlertsResponse)

				assert.EqualValues(t, 2, len(res.Annotations))

				expected := []int{19, 15}
				for i, m := range res.Annotations {
					verifyAnnotated(expected[i], m)
				}
			},
		},
		{
			Name:    "サイレント",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/alerts",
			Query:   func(q url.Values) {
				q.Add("measurements", "1")
				q.Add("measurements", "2")
				q.Add("measurements", "3")
				q.Add("measurements", "4")
				q.Add("measurements", "5")
			},
			Prepare: func(req *http.Request) {
				F.Truncate(db, "measurement_alert")

				_, err := db.Exec(`UPDATE annotated_event SET is_closed = false`)
				assert.NoError(t, err)
				_, err = db.Exec(`UPDATE measurement SET is_closed = false`)
				assert.NoError(t, err)

				// 計測1 -> 3(0m), 7(50m), 14(160m)
				// 計測2 -> 5(20m), 8(60m), 13(140m)
				now := time.Now()

				// 計測1を現在サイレント状態。
				F.Insert(db, model.MeasurementAlert{}, 0, 1, func(i int, r F.Record) {
					r["MeasurementId"] = 1
					r["SilentFrom"] = now.Add(-time.Minute)
					r["SilentUntil"] = now.Add(time.Minute)
				})

				// 計測2の過去、未来にサイレント。
				// -5~-3, 3~5
				F.Insert(db, model.MeasurementAlert{}, 0, 2, func(i int, r F.Record) {
					r["MeasurementId"] = 2
					r["SilentFrom"] = now.Add(time.Minute*time.Duration((i-1)*8-5))
					r["SilentUntil"] = now.Add(time.Minute*time.Duration((i-1)*8-3))
				})

				// 計測3を現在サイレント状態。
				F.Insert(db, model.MeasurementAlert{}, 0, 1, func(i int, r F.Record) {
					r["MeasurementId"] = 3
					r["SilentFrom"] = now.Add(-time.Minute)
					r["SilentUntil"] = now.Add(time.Minute)
				})
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &collectAlertsResponse{}).(*collectAlertsResponse)

				assert.EqualValues(t, 2, len(res.Annotations))

				expected := []int{19, 15}
				for i, m := range res.Annotations {
					verifyAnnotated(expected[i], m)
				}
			},
		},
		{
			Name:    "計測指定なし",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/alerts",
			Prepare: func(req *http.Request) {
				F.Truncate(db, "measurement_alert")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		verifyAnnotated = prepareAlerts(t, db, beginTime)
	})
}