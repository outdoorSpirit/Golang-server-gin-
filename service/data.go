package service

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/gorp.v2"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/resource/influxdb"
)

type DataService struct {
	*Service
	DB *gorp.DbMap
	Influx lib.InfluxDBClient
}

type DataTxService struct {
	*Service
	DB *gorp.Transaction
	Influx lib.InfluxDBClient
}

// ある計測内の指定種別のデータを古い順に取得する。
func (s *DataService) ListCTGData(
	measurementId int,
	measurementType C.MeasurementType,
	begin *time.Time,
	end *time.Time,
) ([]*model.SensorValue, error) {
	var measurement *model.Measurement

	if r, e := rds.InquireMeasurement(s.DB, measurementId); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"measurement_not_found",
			fmt.Sprintf("Measurement %d is not found", measurementId),
			map[string]interface{}{},
		)
	} else {
		measurement = r
	}

	if begin == nil {
		if measurement.FirstTime != nil {
			begin = measurement.FirstTime
		} else {
			zero := time.Unix(0, 0)
			begin = &zero
		}
	}

	if end == nil {
		if measurement.LastTime != nil {
			// 右閉の秒単位検索となるため、1秒進めて最後のデータを含める。
			last := measurement.LastTime.Add(time.Second) 
			end = &last
		} else {
			// 過去のデータしか存在しえない。
			infinity := time.Now()
			end = &infinity
		}
	}

	results := []*model.SensorValue{}

	switch measurementType {
	case C.MeasurementTypeHeartRate:
		if values, e := influxdb.ListHeartRate(s.Influx, measurementId, *begin, *end); e != nil {
			return nil, C.INFLUXDB_OPERATION_ERROR(e)
		} else {
			for _, v := range values {
				results = append(results, &model.SensorValue{v.Value, int64(v.Timestamp.UnixNano() / 1000000)})
			}
		}
	case C.MeasurementTypeTOCO:
		if values, e := influxdb.ListTOCO(s.Influx, measurementId, *begin, *end); e != nil {
			return nil, C.INFLUXDB_OPERATION_ERROR(e)
		} else {
			for _, v := range values {
				results = append(results, &model.SensorValue{v.Value, int64(v.Timestamp.UnixNano() / 1000000)})
			}
		}
	}

	return results, nil
}

type CTGData struct {
	MachineId string `json:"Machine ID"`
	PatientId string `json:"Patient ID"`
	FHR1      string `json:"FHR1"`
	UC        string `json:"UC"`
	FHR2      string `json:"FHR2"`
	Timestamp string `json:"Timestamp"`
}

func (d CTGData) GetFHR1() (int, error) {
	return strconv.Atoi(d.FHR1)
}

func (d CTGData) GetUC() (int, error) {
	return strconv.Atoi(d.UC)
}

func (d CTGData) GetTimestamp() (int64, error) {
	return strconv.ParseInt(d.Timestamp, 10, 64)
}

type CTGRegistrationStats struct {
	SuccessFHR1 int
	FailureFHR1 []error
	SuccessUC   int
	FailureUC   []error
}

// json配列形式の心拍もしくはTOCOデータを登録する。
//
// json内の患者IDが端末IDを含む、すなわち患者IDが同じならば同じ端末になることを前提として計測記録を残す。
// 同じ患者IDが違う端末IDと紐づいている場合も、InfluxDBにはそのまま保存する。
// 患者IDが計測コードとなり、新規ならば新たな計測記録レコードを作成する。
// アプリケーション上の検索は、この計測記録をベースに行うため、json内の端末IDを用いたInfluxDBへのクエリは利用しないこととする。
// この取り決めにより、患者IDと端末IDの組み合わせの不整合は、InfluxDB内には残るがRDBを通じたアプリケーションからは隠蔽されることとなる。
func (s *DataTxService) RegisterCTGData(hospitalId int, ctgData []CTGData) (*CTGRegistrationStats, error) {
	stats := &CTGRegistrationStats{0, []error{}, 0, []error{}}

	if len(ctgData) == 0 {
		return stats, nil
	}

	heartrateRecords := []lib.Point{}
	tocoRecords := []lib.Point{}

	type patientEntry struct {
		machine string
		begin   time.Time
		end     time.Time
	}

	// 患者コードを一意にしたリスト。
	codes := []string{}
	// 患者コードからエントリへのマップ。
	entries := map[string]*patientEntry{}

	for _, data := range ctgData {
		if fhr1, e := data.GetFHR1(); e != nil {
			return nil, C.NewBadRequestError(
				"Invalid FHR1",
				fmt.Sprintf("FHR1 must be in formatted as an integer but '%s'", data.FHR1),
				map[string]interface{}{},
			)
		} else if uc, e := data.GetUC(); e != nil {
			return nil, C.NewBadRequestError(
				"Invalid UC",
				fmt.Sprintf("UC must be in formatted as an integer but '%s'", data.UC),
				map[string]interface{}{},
			)
		} else if ts, e := data.GetTimestamp(); e != nil {
			return nil, C.NewBadRequestError(
				"Invalid Timestamp",
				fmt.Sprintf("Timestamp must be in formatted as an integer but '%s'", data.Timestamp),
				map[string]interface{}{},
			)
		} else {
			observedAt := time.Unix(ts / 1000, (ts % 1000) * 1000000)

			heartrateRecords = append(heartrateRecords, &model.HeartRate{
				0, data.PatientId, data.MachineId, fhr1, observedAt,
			})
			tocoRecords = append(tocoRecords, &model.TOCO{
				0, data.PatientId, data.MachineId, uc, observedAt,
			})

			if d, be := entries[data.PatientId]; be {
				if observedAt.After(d.end) {
					d.end = observedAt
				}
				if observedAt.Before(d.begin) {
					d.begin = observedAt
				}
			} else {
				entries[data.PatientId] = &patientEntry{data.MachineId, observedAt, observedAt}
				codes = append(codes, data.PatientId)
			}
		}
	}

	now := time.Now()

	// 既存の計測を取得。
	measurementMap := map[string]*model.Measurement{}

	if measurements, e := rds.InquireMeasurementsByCodes(s.DB, hospitalId, codes); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		for _, m := range measurements {
			measurementMap[m.Code] = m
		}
	}

	// 新規患者登録。
	// 新規の計測は新規の患者を生成し、patientモデルは新規計測作成時にしか利用しないため、既存のpatientで埋める必要はない。
	patientMap := map[string]*model.Patient{}

	newPatients := []interface{}{}

	for _, c := range codes {
		// 現状では、新規の計測は必ず新規の患者を生成する。
		if _, be := measurementMap[c]; !be {
			p := &model.Patient{
				HospitalId: hospitalId,
				CreatedAt:  now,
				ModifiedAt: now,
			}

			patientMap[c] = p
			newPatients = append(newPatients, p)
		}
	}

	if e := s.DB.Insert(newPatients...); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	// 新規端末登録。
	terminalMap := map[string]*model.MeasurementTerminal{}
	terminalCodes := []string{}

	for _, c := range codes {
		entry := entries[c]
		terminalMap[entry.machine] = nil
		terminalCodes = append(terminalCodes, entry.machine)
	}

	if terminals, e := rds.InquireTerminalsByCodes(s.DB, hospitalId, terminalCodes); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		for _, t := range terminals {
			terminalMap[t.Code] = t
		}
	}

	newTerminals := []interface{}{}

	for _, c := range terminalCodes {
		// モデルがセットされていないものが新規端末。
		if terminalMap[c] == nil {
			t := &model.MeasurementTerminal{
				HospitalId:   hospitalId,
				Code:         c,
				DisplayOrder: 0,
				CreatedAt:    now,
				ModifiedAt:   now,
			}

			terminalMap[c] = t
			newTerminals = append(newTerminals, t)
		}
	}

	if e := s.DB.Insert(newTerminals...); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	// 新規計測ならば登録。
	newMeasurements := []interface{}{}
	updMeasurements := []interface{}{}

	for _, c := range codes {
		entry := entries[c]
		if m, be := measurementMap[c]; be {
			if m.FirstTime == nil || m.FirstTime.After(entry.begin) {
				m.FirstTime = &entry.begin
			}
			if m.LastTime == nil || m.LastTime.Before(entry.end) {
				m.LastTime = &entry.end
			}
			m.ModifiedAt = now
			updMeasurements = append(updMeasurements, m)
		} else {
			m := &model.Measurement{
				Code:       c,
				PatientId:  patientMap[c].Id,
				TerminalId: terminalMap[entry.machine].Id,
				FirstTime:  &entry.begin,
				LastTime:   &entry.end,
				CreatedAt:  now,
				ModifiedAt: now,
			}
			measurementMap[c] = m
			newMeasurements = append(newMeasurements, m)
		}
	}

	if e := s.DB.Insert(newMeasurements...); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	if _, e := s.DB.Update(updMeasurements...); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	// InfluxDB用モデルに計測記録IDを移す。
	for _, r := range heartrateRecords {
		p := r.(*model.HeartRate)
		p.MeasurementId = measurementMap[p.PatientCode].Id
	}

	for _, r := range tocoRecords {
		p := r.(*model.TOCO)
		p.MeasurementId = measurementMap[p.PatientCode].Id
	}

	// InfluxDBに登録。
	errors := s.Influx.Insert("spiker", heartrateRecords...)

	stats.SuccessFHR1 = len(heartrateRecords) - len(errors)
	stats.FailureFHR1 = append(stats.FailureFHR1, errors...)

	errors = s.Influx.Insert("spiker", tocoRecords...)

	stats.SuccessUC = len(tocoRecords) - len(errors)
	stats.FailureUC = append(stats.FailureUC, errors...)

	return stats, nil
}