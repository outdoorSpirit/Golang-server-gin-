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

	"github.com/stretchr/testify/assert"

	//C "github.com/spiker/spiker-server/constant"
	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestMonitorData_ListHeartRate(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)
	influx := lib.GetInfluxDB()

	beginTime := time.Date(2021, time.January, 2, 3, 4, 5, 0, time.UTC)

	httpTests := test.HttpTests{
		{
			Name:    "心拍全データ取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/heartrates",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listHeartRateResponse{}).(*listHeartRateResponse)

				assert.EqualValues(t, 250, len(res.Records))

				for i, r := range res.Records {
					assert.EqualValues(t, 1001+(i*4), r.Value)
					assert.EqualValues(t, beginTime.Add(time.Duration(1+i*4)*time.Second).UnixNano() / 1000000, r.ObservedAt)
				}
			},
		},
		{
			Name:    "心拍期間指定取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/heartrates",
			Query:   func(q url.Values) {
				q.Add("minutes", "5")
				q.Add("end", "2021-01-02T03:10:00+00:00")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listHeartRateResponse{}).(*listHeartRateResponse)

				// 先頭:+55秒, 末尾:+355秒 -> +57,+61,...,+353
				// (353-57)/4+1 = 296/4+1 = 75
				assert.EqualValues(t, 75, len(res.Records))

				for i, r := range res.Records {
					assert.EqualValues(t, 1057+(i*4), r.Value)
					assert.EqualValues(t, beginTime.Add(time.Duration(57+i*4)*time.Second).UnixNano() / 1000000, r.ObservedAt)
				}
			},
		},
		{
			Name:    "計測が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/0/heartrates",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/heartrates",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement_terminal", "patient", "measurement")

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

		assert.NoError(t, influx.Delete("spiker", time.Unix(0, 0), time.Now().Add(time.Duration(24*365*100)*time.Hour), ""))

		points := []lib.Point{}

		for i := 0; i < 1000; i++ {
			points = append(points, &model.HeartRate{
				MeasurementId: measurements[i%4].Id,
				PatientCode: "p",
				MachineCode: "m",
				Value: 1000+i,
				Timestamp: beginTime.Add(time.Duration(i)*time.Second),
			})
			points = append(points, &model.TOCO{
				MeasurementId: measurements[i%4].Id,
				PatientCode: "p",
				MachineCode: "m",
				Value: 3000+i,
				Timestamp: beginTime.Add(time.Duration(i)*time.Second),
			})
		}

		influx.Insert("spiker", points...)
	})
}

func TestMonitorData_ListTOCO(t *testing.T) {
	auth := (&F.MonitorFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)
	influx := lib.GetInfluxDB()

	beginTime := time.Date(2021, time.January, 2, 3, 4, 5, 0, time.UTC)

	httpTests := test.HttpTests{
		{
			Name:    "TOCO全データ取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/tocos",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listTOCOResponse{}).(*listTOCOResponse)

				assert.EqualValues(t, 250, len(res.Records))

				for i, r := range res.Records {
					assert.EqualValues(t, 3001+(i*4), r.Value)
					assert.EqualValues(t, beginTime.Add(time.Duration(1+i*4)*time.Second).UnixNano() / 1000000, r.ObservedAt)
				}
			},
		},
		{
			Name:    "TOCO期間指定取得",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/2/tocos",
			Query:   func(q url.Values) {
				q.Add("minutes", "5")
				q.Add("end", "2021-01-02T03:10:00+00:00")
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &listTOCOResponse{}).(*listTOCOResponse)

				// 先頭:+55秒, 末尾:+355秒 -> +57,+61,...,+353
				// (353-57)/4+1 = 296/4+1 = 75
				assert.EqualValues(t, 75, len(res.Records))

				for i, r := range res.Records {
					assert.EqualValues(t, 3057+(i*4), r.Value)
					assert.EqualValues(t, beginTime.Add(time.Duration(57+i*4)*time.Second).UnixNano() / 1000000, r.ObservedAt)
				}
			},
		},
		{
			Name:    "計測が無い",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/0/tocos",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			Name:    "病院が違う",
			Method:  http.MethodGet,
			Token:   auth.Token(0),
			Path:    "/1/measurements/4/tocos",
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, func() {
		F.Truncate(db, "measurement_terminal", "patient", "measurement")

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

		assert.NoError(t, influx.Delete("spiker", time.Unix(0, 0), time.Now().Add(time.Duration(24*365*100)*time.Hour), ""))

		points := []lib.Point{}

		for i := 0; i < 1000; i++ {
			points = append(points, &model.HeartRate{
				MeasurementId: measurements[i%4].Id,
				PatientCode: "p",
				MachineCode: "m",
				Value: 1000+i,
				Timestamp: beginTime.Add(time.Duration(i)*time.Second),
			})
			points = append(points, &model.TOCO{
				MeasurementId: measurements[i%4].Id,
				PatientCode: "p",
				MachineCode: "m",
				Value: 3000+i,
				Timestamp: beginTime.Add(time.Duration(i)*time.Second),
			})
		}

		influx.Insert("spiker", points...)
	})
}
