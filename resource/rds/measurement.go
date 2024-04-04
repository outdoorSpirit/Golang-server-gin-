package rds

import (
	"database/sql"
	//"encoding/json"
	"fmt"
	//"strings"
	"time"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
)

func InquireMeasurement(
	db model.QueryExecutor,
	id int,
) (*model.Measurement, error) {
	if r, e := db.Get(model.Measurement{}, id); e != nil {
		return nil, e
	} else if r == nil {
		return nil, nil
	} else {
		return r.(*model.Measurement), nil
	}
}

func LockMeasurement(
	db model.QueryExecutor,
	id int,
) (*model.Measurement, error) {
	record := model.Measurement{}

	if e := db.SelectOne(&record, `SELECT * FROM measurement WHERE id = $1 FOR UPDATE`, id); e != nil {
		if e == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, e
		}
	} 

	return &record, nil
}

// 病院と計測コードから計測記録を取得する。
func InquireMeasurementsByCodes(
	db model.QueryExecutor,
	hospitalId int,
	codes []string,
) ([]*model.Measurement, error) {
	query := `SELECT
		m.*
	FROM
		measurement AS m
		INNER JOIN patient AS p ON m.patient_id = p.id
	WHERE
		m.code IN (:codes) AND p.hospital_id = :hospital_id`

	records := []*model.Measurement{}

	if _, e := db.Select(&records, query, map[string]interface{}{
		"codes": codes,
		"hospital_id": hospitalId,
	}); e != nil {
		return nil, e
	}

	return records, nil
}

// 患者もしくは端末に関する計測記録一覧を取得する。
// TODO 医師によるアクセス制限を必要に応じて追加。
func ListMeasurements(
	db model.QueryExecutor,
	hospitalId int,
	patientId *int,
	terminalId *int,
	limit int,
	offset int,
) ([]*model.MeasurementEntity, int64, error) {
	ip := incrementalPlaceholder{0}

	// REVIEW 患者側から病院との関連を取る。
	q := andQuery().add(fmt.Sprintf("p.hospital_id = $%d", ip.GetIndex()), hospitalId)

	if patientId != nil {
		q.add(fmt.Sprintf("m.patient_id = $%d", ip.GetIndex()), *patientId)
	} else if terminalId != nil {
		q.add(fmt.Sprintf("m.terminal_id = $%d", ip.GetIndex()), *terminalId)
	}

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s, %s
		FROM
			measurement AS m
			INNER JOIN patient AS p ON m.patient_id = p.id
			INNER JOIN measurement_terminal AS t ON m.terminal_id = t.id
		%s
		ORDER BY
			m.created_at DESC
		LIMIT $%d OFFSET $%d`,
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.Patient{}, "p", "p"),
		prefixColumns(model.MeasurementTerminal{}, "t", "t"),
		where, ip.GetIndex(), ip.GetIndex(),
	)

	results := []*model.MeasurementEntity{}

	if rows, e := db.Query(query, params.clone().add(limit, offset).values...); e != nil {
		return nil, 0, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			entity := model.MeasurementEntity{
				Measurement: &model.Measurement{},
				Terminal:    &model.MeasurementTerminal{},
				Patient:     &model.Patient{},
			}

			scanRows(db, rows, entity.Measurement, "m")
			scanRows(db, rows, entity.Terminal, "t")
			scanRows(db, rows, entity.Patient, "p")

			results = append(results, &entity)
		})
	}

	if total, e := db.SelectInt(fmt.Sprintf(`SELECT
			COUNT(*)
		FROM
			measurement AS m
			INNER JOIN patient AS p ON m.patient_id = p.id
			INNER JOIN measurement_terminal AS t ON m.terminal_id = t.id
		%s`, where), params.values...); e != nil {
		return nil, 0, e
	} else {
		return results, total, nil
	}
}

// IDから計測記録を取得する。
func FetchMeasurment(
	db model.QueryExecutor,
	id int,
) (*model.MeasurementEntity, error) {
	query := fmt.Sprintf(
		`SELECT
			%s, %s, %s
		FROM
			measurement AS m
			INNER JOIN patient AS p ON m.patient_id = p.id
			INNER JOIN measurement_terminal AS t ON m.terminal_id = t.id
		WHERE
			m.id = $1`,
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.Patient{}, "p", "p"),
		prefixColumns(model.MeasurementTerminal{}, "t", "t"),
	)

	if rows, e := db.Query(query, id); e != nil {
		return nil, e
	} else {
		var entity *model.MeasurementEntity = nil

		safeRowsIterator(rows, func(rows *sql.Rows) {
			entity = &model.MeasurementEntity{
				Measurement: &model.Measurement{},
				Terminal:    &model.MeasurementTerminal{},
				Patient:     &model.Patient{},
			}

			scanRows(db, rows, entity.Measurement, "m")
			scanRows(db, rows, entity.Terminal, "t")
			scanRows(db, rows, entity.Patient, "p")
		})

		return entity, nil
	}
}

func CheckMeasurementAccessByDoctor(
	db model.QueryExecutor,
	doctorId int,
	measurementId int,
) (bool, error) {
	// TODO 医師と患者の関連への切り替え。
	query := `SELECT
			m.*
		FROM
			measurement AS m
			INNER JOIN patient AS p ON m.patient_id = p.id
			INNER JOIN doctor AS d ON p.hospital_id = d.hospital_id
		WHERE
			m.id = $1 AND d.id = $2`

	record := model.Measurement{}

	if e := db.SelectOne(&record, query, measurementId, doctorId); e != nil {
		if e == sql.ErrNoRows {
			return false, nil
		} else {
			return false, e
		}
	} else {
		return true, nil
	}
}

func CheckMeasurementAccessByHospital(
	db model.QueryExecutor,
	hospitalId int,
	measurementId int,
) (bool, error) {
	// TODO 医師と患者の関連への切り替え。
	query := `SELECT
			m.*
		FROM
			measurement AS m
			INNER JOIN patient AS p ON m.patient_id = p.id
		WHERE
			m.id = $1 AND p.hospital_id = $2`

	record := model.Measurement{}

	if e := db.SelectOne(&record, query, measurementId, hospitalId); e != nil {
		if e == sql.ErrNoRows {
			return false, nil
		} else {
			return false, e
		}
	} else {
		return true, nil
	}
}

// 検査対象の計測を取得する。
// 全体でduration以上の長さのデータが存在し、interval未満の時間に前回の診断が存在しない計測が対象。
// 返すエンティティは、診断用データは持たず、フィールドもセットされない。
func CollectMeasurementsForAssessment(
	db model.QueryExecutor,
	duration time.Duration,
	interval time.Duration,
	diagnosisTime time.Time,
) ([]*model.DiagnosisMeasurmentEntity, error) {
	query := fmt.Sprintf(
		`SELECT
			%s, %s
		FROM
			measurement AS m
			LEFT JOIN (
				SELECT
					*
				FROM
					(
						SELECT
							d_.id, d_.measurement_id,
							RANK() OVER (PARTITION BY d_.measurement_id ORDER BY d_.range_until DESC, d_.id DESC) AS rank
						FROM
							diagnosis AS d_
							INNER JOIN measurement AS m_ ON d_.measurement_id = m_.id
						WHERE
							m_.last_time >= $1
					) AS r
				WHERE
					r.rank = 1
			) AS ld ON m.id = ld.measurement_id
			LEFT JOIN diagnosis AS d ON ld.id = d.id
		WHERE
			m.last_time IS NOT NULL AND m.first_time IS NOT NULL AND (
				m.last_time >= $1
				AND m.first_time <= $1
				AND m.last_time - m.first_time >= interval '1 second' * $2
				AND COALESCE(d.range_until <= $3, true)
				AND COALESCE(d.range_until < m.last_time, true)
			)
		ORDER BY m.id ASC`,
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.Diagnosis{}, "d", "d"),
	)

	if rows, e := db.Query(
		query,
		diagnosisTime.Add(-duration),
		duration.Seconds(),
		diagnosisTime.Add(-interval),
	); e != nil {
		return nil, e
	} else {
		results := []*model.DiagnosisMeasurmentEntity{}

		safeRowsIterator(rows, func(rows *sql.Rows) {
			latestDiagnosis := &model.Diagnosis{}

			r := &model.DiagnosisMeasurmentEntity{
				Measurement: &model.Measurement{},
				LatestDiagnosis: nil,
			}

			scanRows(db, rows, r.Measurement, "m")
			scanRows(db, rows, latestDiagnosis, "d")

			if latestDiagnosis.Id > 0 {
				r.LatestDiagnosis = latestDiagnosis
			}

			results = append(results, r)
		})

		return results, nil
	}
}

// 計測のサイレント状態を取得する。
func GetSilent(
	db model.QueryExecutor,
	id int,
	now time.Time,
) (*model.MeasurementAlert, error) {
	records := []*model.MeasurementAlert{}

	if _, e := db.Select(&records,
		`SELECT
			*
		FROM
			measurement_alert
		WHERE
			measurement_id = $1 AND silent_from <= $2 AND silent_until >= $3`,
		id, now, now,
	); e != nil {
		return nil, e
	}

	if len(records) > 0 {
		return records[len(records)-1], nil
	} else {
		return nil, nil
	}
}

// 計測をサイレント状態にする
func SetSilent(
	db model.QueryExecutor,
	id int,
	now time.Time,
	duration time.Duration,
) (*model.MeasurementAlert, error) {
	if alert, e := GetSilent(db, id, now); e != nil {
		return nil, e
	} else if alert != nil {
		return nil, nil
	}

	record := &model.MeasurementAlert{
		MeasurementId: id,
		SilentFrom: now,
		SilentUntil: now.Add(duration),
		Memo: "",
		CreatedAt: now,
		ModifiedAt: now,
	}

	if e := db.Insert(record); e != nil {
		return nil, e
	}

	return record, nil
}

// 現在サイレント状態ならば、それを解除する。
func StopCurrentSilent(
	db model.QueryExecutor,
	id int,
	now time.Time,
) error {
	query := `UPDATE measurement_alert
		SET silent_until = $1, modified_at = $1
		WHERE measurement_id = $2 AND silent_from <= $1 AND silent_until >= $1`

	if _, e := db.Exec(query, now, id); e != nil {
		return e
	}

	return nil
}