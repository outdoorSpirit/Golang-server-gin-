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

func InquirePatient(
	db model.QueryExecutor,
	id int,
) (*model.Patient, error) {
	// REVIEW 現状やることは同じ。
	return FetchPatient(db, id)
}

// 患者一覧を取得する。
// begin, endが指定されている場合、その期間に計測を持つ患者に限定する。
// 計測のある患者を優先し、最終計測日時が最も遅い順にソートする。
func ListPatients(
	db model.QueryExecutor,
	hospitalId int,
	begin *time.Time,
	end *time.Time,
	limit int,
	offset int,
) ([]*model.Patient, int64, error) {
	ip := incrementalPlaceholder{0}

	q := andQuery().add(fmt.Sprintf("p.hospital_id = $%d", ip.GetIndex()), hospitalId)

	if begin != nil {
		q.add(fmt.Sprintf("pm.last >= $%d", ip.GetIndex()), *begin)
	}
	if end != nil {
		q.add(fmt.Sprintf("pm.first <= $%d", ip.GetIndex()), *end)
	}

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s
		FROM
			patient AS p
			LEFT JOIN (
				SELECT
					p_.id, MIN(m.first_time) AS first, MAX(m.last_time) AS last
				FROM
					patient AS p_
					INNER JOIN measurement AS m ON p_.id = m.patient_id
				WHERE
					p_.hospital_id = $1
				GROUP BY
					p_.id
			) AS pm ON p.id = pm.id
		%s
		ORDER BY
			pm.last DESC NULLS LAST, p.id ASC
		LIMIT $%d OFFSET $%d`,
		prefixColumns(model.Patient{}, "p", "p"),
		where, ip.GetIndex(), ip.GetIndex(),
	)

	records := []*model.Patient{}

	if rows, e := db.Query(query, params.clone().add(limit, offset).values...); e != nil {
		return nil, 0, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			patient := &model.Patient{}

			scanRows(db, rows, patient, "p")

			records = append(records, patient)
		})
	}

	totalQuery := fmt.Sprintf(
		`SELECT
			COUNT(*)
		FROM
			patient AS p
			LEFT JOIN (
				SELECT
					p_.id, MIN(m.first_time) AS first, MAX(m.last_time) AS last
				FROM
					patient AS p_
					INNER JOIN measurement AS m ON p_.id = m.patient_id
				WHERE
					p_.hospital_id = $1
				GROUP BY
					p_.id
			) AS pm ON p.id = pm.id
		%s`,
		where,
	)

	if total, e := db.SelectInt(totalQuery, params.values...); e != nil {
		return nil, 0, e
	} else {
		return records, total, nil
	}
}

// 患者情報を取得する。
func FetchPatient(
	db model.QueryExecutor,
	id int,
) (*model.Patient, error) {
	query := fmt.Sprintf(`SELECT %s FROM patient AS p WHERE id = $1`, prefixColumns(model.Patient{}, "p", "p"))

	if rows, e := db.Query(query, id); e != nil {
		return nil, e
	} else {
		var record *model.Patient

		safeRowsIterator(rows, func(rows *sql.Rows) {
			record = &model.Patient{}

			scanRows(db, rows, record, "p")
		})

		return record, nil
	}
}