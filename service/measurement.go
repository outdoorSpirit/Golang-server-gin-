package service

import (
	"fmt"
	"time"

	"gopkg.in/gorp.v2"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/resource/influxdb"
)

type MeasurementService struct {
	*Service
	DB *gorp.DbMap
	Influx lib.InfluxDBClient
}

type MeasurementTxService struct {
	*Service
	DB *gorp.Transaction
	Influx lib.InfluxDBClient
}

// 医師からのアクセス可否を調べる。
func (s *MeasurementService) CheckAccessByDoctor(
	me *model.HospitalDoctor,
	id int,
) error {
	if ok, e := rds.CheckMeasurementAccessByDoctor(s.DB, me.Doctor.Id, id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if !ok {
		return C.NewNotFoundError(
			"measurement_not_found",
			fmt.Sprintf("Measurement %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// 患者に関する計測記録一覧を取得する。
// TODO 医師によるアクセス制限を必要に応じて追加。
func (s *MeasurementService) List(
	hospitalId int,
	patientId *int,
	terminalId *int,
	limit int,
	offset int,
) ([]*model.MeasurementEntity, int64, error) {
	if r, t, e := rds.ListMeasurements(s.DB, hospitalId, patientId, terminalId, limit, offset); e != nil {
		return nil, 0, C.DB_OPERATION_ERROR(e)
	} else {
		return r, t, nil
	}
}

// 病院内の計測をコードから取得する。
func (s *MeasurementService) InquireByCode(
	hospitalId int,
	code string,
) (*model.Measurement, error) {
	if ms, e := rds.InquireMeasurementsByCodes(s.DB, hospitalId, []string{code}); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if len(ms) == 0 {
		return nil, C.NewNotFoundError(
			"measurement_not_found",
			fmt.Sprintf("Measurement %s is not found in hospital %d", code, hospitalId),
			map[string]interface{}{},
		)
	} else {
		return ms[0], nil
	}
}

// 病院からのアクセスを調べる。
func (s *MeasurementService) CheckAccessByHospital(
	hospitalId int,
	id int,
) error {
	if ok, e := rds.CheckMeasurementAccessByHospital(s.DB, hospitalId, id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if !ok {
		return C.NewNotFoundError(
			"measurement_not_found",
			fmt.Sprintf("Measurement %d is not found in hospital %d", id, hospitalId),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// IDから計測記録を取得する。
func (s *MeasurementService) Fetch(
	id int,
) (*model.MeasurementEntity, error) {
	if r, e := rds.FetchMeasurment(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"measurement_not_found",
			fmt.Sprintf("Measurement %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// 検査対象の計測を取得する。
func (s *MeasurementService) CollectForAssessment(
	diagnosisTime time.Time,
	duration time.Duration,
	interval time.Duration,
) ([]*model.DiagnosisMeasurmentEntity, error) {
	measurements := []*model.DiagnosisMeasurmentEntity{}

	if ms, e := rds.CollectMeasurementsForAssessment(s.DB, duration, interval, diagnosisTime); e != nil {
		return nil, e
	} else {
		measurements = ms
	}

	// それぞれの計測のduration分のデータを取得する。
	for _, m := range measurements {
		if vs, e := influxdb.ListHeartRate(s.Influx, m.Id, diagnosisTime.Add(-duration), diagnosisTime); e != nil {
			return nil, e
		} else {
			m.HeartRates = vs
		}

		if vs, e := influxdb.ListTOCO(s.Influx, m.Id, diagnosisTime.Add(-duration), diagnosisTime); e != nil {
			return nil, e
		} else {
			m.TOCOs = vs
		}
	}

	return measurements, nil
}

// 計測のサイレント状態を取得する。
func (s *MeasurementService) GetSilent(
	id int,
) (*model.MeasurementAlert, error) {
	now := time.Now()

	if alert, e := rds.GetSilent(s.DB, id, now); e != nil {
		return nil, e
	} else {
		return alert, nil
	}
}

// 医師からの更新可否を調べる。
func (s *MeasurementTxService) CheckUpdateByDoctor(
	me *model.HospitalDoctor,
	id int,
) error {
	if ok, e := rds.CheckMeasurementAccessByDoctor(s.DB, me.Doctor.Id, id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if !ok {
		return C.NewNotFoundError(
			"measurement_not_found",
			fmt.Sprintf("Measurement %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// 計測をサイレント状態にする
func (s *MeasurementTxService) SetSilent(
	id int,
) (*model.MeasurementAlert, error) {
	if _, e := rds.LockMeasurement(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	now := time.Now()

	if r, e := rds.SetSilent(s.DB, id, now, C.AlertSilentDuration); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewBadRequestError(
			"measurement_already_silent",
			fmt.Sprintf("Measurement #%d is already silent", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// 計測が存在しないことを保証する。
// 計測コードが既に存在する場合、removeIfExistsがtrueの場合削除を行い、falseならばエラーを返す。
func (s *MeasurementTxService) EnsureNotExists(
	hospitalId int,
	code string,
	removeIfExists bool,
) error {
	// 既存の計測を削除。
	if ms, e := rds.InquireMeasurementsByCodes(s.DB, hospitalId, []string{code}); e != nil {
		return e
	} else if len(ms) > 1 {
		return fmt.Errorf("%d measurements whose codes are '%s' are found in hospital #%d", len(ms), code, hospitalId)
	} else if len(ms) == 1 {
		if !removeIfExists {
			return fmt.Errorf("Measurement code '%s' already exists in hospital #%d", code, hospitalId)
		}

		if _, e := s.DB.Delete(ms[0]); e != nil {
			return e
		}

		// 計測IDに対する全てのデータを削除する。
		if e := s.Influx.DeleteAll("spiker", fmt.Sprintf(`measurement_id="%d"`, ms[0].Id)); e != nil {
			return e
		}
	}

	return nil
}

// 計測を完了状態にする。
func (s *MeasurementTxService) Close(
	id int,
	memo string,
) error {
	mm, err := rds.InquireMeasurement(s.DB, id)

	if err != nil {
		return C.DB_OPERATION_ERROR(err)
	} else if mm == nil {
		return C.NewNotFoundError(
			"measurement_not_found",
			fmt.Sprintf("Measurement %d is not found", id),
			map[string]interface{}{},
		)
	}

	now := time.Now()

	mm.IsClosed = true
	mm.ClosingMemo = &memo
	mm.ClosedAt = &now
	mm.ModifiedAt = now

	if _, e := s.DB.Update(mm); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	return nil
}