package rds

import (
	"database/sql"
	//"encoding/json"
	"fmt"
	//"strings"
	"time"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
)

func InquireDiagnosis(
	db model.QueryExecutor,
	id int,
) (*model.Diagnosis, error) {
	if r, e := db.Get(model.Diagnosis{}, id); e != nil {
		return nil, e
	} else if r == nil {
		return nil, nil
	} else {
		return r.(*model.Diagnosis), nil
	}
}

func FetchAlgorithmByName(
	db model.QueryExecutor,
	name string,
	version string,
) (*model.DiagnosisAlgorithm, error) {
	result := model.DiagnosisAlgorithm{}

	if e := db.SelectOne(&result, `SELECT * FROM diagnosis_algorithm WHERE name = $1 AND version = $2`, name, version); e != nil {
		if e == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, e
		}
	}

	return &result, nil
}

// 診断から計測、病院への関連があるか調べる。
func TraceDiagnosisRelations(
	db model.QueryExecutor,
	id int,
	measurementId int,
	hospitalId int,
) (bool, error) {
	query := fmt.Sprintf(
		`SELECT
			*
		FROM
			diagnosis AS d
			INNER JOIN measurement AS m ON d.measurement_id = m.id
			INNER JOIN patient AS p ON m.patient_id = p.id
		WHERE
			d.id = $1 AND d.measurement_id = $2 AND p.hospital_id = $3`,
	)

	if rows, e := db.Query(query, id, measurementId, hospitalId); e != nil {
		return false, e
	} else {
		defer rows.Close()
		return rows.Next(), nil
	}
}

// 計測における全ての診断を新しい順に取得する。
func ListDiagnoses(
	db model.QueryExecutor,
	measurementId int,
) ([]*model.DiagnosisEntity, error) {
	ip := incrementalPlaceholder{0}

	q := andQuery().add(fmt.Sprintf("d.measurement_id = $%d", ip.GetIndex()), measurementId)

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s
		FROM
			diagnosis AS d
			LEFT JOIN computed_diagnosis AS cd ON d.id = cd.diagnosis_id
			LEFT JOIN diagnosis_algorithm AS a ON cd.algorithm_id = a.id
		%s
		ORDER BY
			d.created_at DESC`,
		prefixColumns(model.Diagnosis{}, "d", "d"),
		prefixColumns(model.DiagnosisAlgorithm{}, "a", "a"),
		where,
	)

	records := []*model.DiagnosisEntity{}

	if rows, e := db.Query(query, params.values...); e != nil {
		return nil, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			entity := model.DiagnosisEntity{
				Diagnosis: &model.Diagnosis{},
				Algorithm: &model.DiagnosisAlgorithm{},
				Contents: []*model.DiagnosisContent{},
			}

			scanRows(db, rows, entity.Diagnosis, "d")
			scanRows(db, rows, entity.Algorithm, "a")

			records = append(records, &entity)
		})
	}

	if len(records) > 0 {
		if e := feedDiagnosisContents(db, records...); e != nil {
			return nil, e
		}
	}

	return records, nil
}

// ある病院において、指定した日時以降の既定値以上のリスクを持つ自動診断を全て取得する。
func ListFollowingDiagnoses(
	db model.QueryExecutor,
	hospitalId int,
	risk int,
	beginTime time.Time,
) ([]*model.DiagnosisEntity, error) {
	// REVIEW リスクが規定値を超える診断項目を、指定日時以降に持つ診断に絞り込む。
	query := fmt.Sprintf(
		`SELECT
			%s, %s
		FROM
			diagnosis AS d
			INNER JOIN measurement AS m ON d.measurement_id = m.id
			INNER JOIN patient AS p ON m.patient_id = p.id
			INNER JOIN computed_diagnosis AS cd ON d.id = cd.diagnosis_id
			INNER JOIN diagnosis_algorithm AS a ON cd.algorithm_id = a.id
		WHERE
			p.hospital_id = $1
			AND (SELECT MAX(risk) FROM diagnosis_content WHERE diagnosis_id = d.id AND range_from >= $2) >= $3
		ORDER BY
			d.range_from ASC`,
		prefixColumns(model.Diagnosis{}, "d", "d"),
		prefixColumns(model.DiagnosisAlgorithm{}, "a", "a"),
	)

	records := []*model.DiagnosisEntity{}

	if rows, e := db.Query(query, hospitalId, beginTime, risk); e != nil {
		return nil, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			entity := model.DiagnosisEntity{
				Diagnosis: &model.Diagnosis{},
				Algorithm: &model.DiagnosisAlgorithm{},
				Contents: []*model.DiagnosisContent{},
			}

			scanRows(db, rows, entity.Diagnosis, "d")
			scanRows(db, rows, entity.Algorithm, "a")

			records = append(records, &entity)
		})
	}

	if len(records) > 0 {
		if e := feedDiagnosisContents(db, records...); e != nil {
			return nil, e
		}
	}

	return records, nil
}

// 診断情報を取得する。
func FetchDiagnosis(
	db model.QueryExecutor,
	id int,
) (*model.DiagnosisEntity, error) {
	query := fmt.Sprintf(
		`SELECT
			%s, %s
		FROM
			diagnosis AS d
			LEFT JOIN computed_diagnosis AS cd ON d.id = cd.diagnosis_id
			LEFT JOIN diagnosis_algorithm AS a ON cd.algorithm_id = a.id
		WHERE
			d.id = $1`,
		prefixColumns(model.Diagnosis{}, "d", "d"),
		prefixColumns(model.DiagnosisAlgorithm{}, "a", "a"),
	)

	var entity *model.DiagnosisEntity

	if rows, e := db.Query(query, id); e != nil {
		return nil, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			entity = &model.DiagnosisEntity{
				Diagnosis: &model.Diagnosis{},
				Algorithm: &model.DiagnosisAlgorithm{},
				Contents: []*model.DiagnosisContent{},
			}

			scanRows(db, rows, entity.Diagnosis, "d")
			scanRows(db, rows, entity.Algorithm, "a")
		})
	}

	if entity != nil {
		if e := feedDiagnosisContents(db, entity); e != nil {
			return nil, e
		}
	}

	return entity, nil
}

// ある計測において、指定日時以前の最新の基線パラメータを取得する。
func FindLatestContent(
	db model.QueryExecutor,
	measurementId int,
	current time.Time,
) (*model.DiagnosisContent, error) {
	query := fmt.Sprintf(
		`SELECT
			dc.*
		FROM
			diagnosis_content AS dc
			INNER JOIN diagnosis AS d ON dc.diagnosis_id = d.id
		WHERE
			d.measurement_id = $1
			AND dc.range_until < $2
			AND dc.parameters ?| array[$3, $4, $5, $6]
		ORDER BY
			dc.range_until DESC
		LIMIT 1`,
	)

	record := model.DiagnosisContent{}

	params := []interface{}{measurementId, current}

	for _, evt := range C.BaselineEvents {
		params = append(params, evt)
	}

	if e := db.SelectOne(&record, query, params...); e != nil {
		if e == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, e
		}
	} else {
		return &record, nil
	}
}

// DiagnosisEntityに診断項目をセットする。
func feedDiagnosisContents(
	db model.QueryExecutor,
	entities ...*model.DiagnosisEntity,
) error {
	ids := []int{}

	for _, e := range entities {
		ids = append(ids, e.Diagnosis.Id)
	}

	query := `SELECT * FROM diagnosis_content AS dc WHERE dc.diagnosis_id IN (:ids) ORDER BY dc.range_from ASC`

	contents := []*model.DiagnosisContent{}

	if _, e := db.Select(&contents, query, map[string]interface{}{
		"ids": ids,
	}); e != nil {
		return e
	}

	contentMap := map[int][]*model.DiagnosisContent{}

	for _, c := range contents {
		if _, be := contentMap[c.DiagnosisId]; !be {
			contentMap[c.DiagnosisId] = []*model.DiagnosisContent{}
		}
		contentMap[c.DiagnosisId] = append(contentMap[c.DiagnosisId], c)
	}

	for _, e := range entities {
		if cs, be := contentMap[e.Diagnosis.Id]; be {
			e.Contents = append(e.Contents, cs...)
		}
	}

	return nil
}