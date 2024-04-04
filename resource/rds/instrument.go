package rds

import (
	"database/sql"
	//"encoding/json"
	"fmt"
	//"strings"
	//"time"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
)

// コードから端末を取得する。
func InquireTerminalsByCodes(
	db model.QueryExecutor,
	hospitalId int,
	codes []string,
) ([]*model.MeasurementTerminal, error) {
	query := `SELECT * FROM measurement_terminal WHERE code IN (:codes) AND hospital_id = :hospital_id`

	records := []*model.MeasurementTerminal{}

	if _, e := db.Select(&records, query, map[string]interface{}{
		"codes": codes,
		"hospital_id": hospitalId,
	}); e != nil {
		return nil, e
	}

	return records, nil
}

func InquireTerminal(
	db model.QueryExecutor,
	id int,
) (*model.MeasurementTerminal, error) {
	// REVIEW 現状やることは同じ。
	return FetchTerminal(db, id)
}

// 計測端末一覧を取得する。
func ListTerminals(
	db model.QueryExecutor,
	hospitalId int,
	limit int,
	offset int,
) ([]*model.MeasurementTerminal, int64, error) {
	ip := incrementalPlaceholder{0}

	q := andQuery().add(fmt.Sprintf("t.hospital_id = $%d", ip.GetIndex()), hospitalId)

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s
		FROM
			measurement_terminal AS t
		%s
		ORDER BY
			t.id ASC
		LIMIT $%d OFFSET $%d`,
		prefixColumns(model.MeasurementTerminal{}, "t", "t"),
		where, ip.GetIndex(), ip.GetIndex(),
	)

	records := []*model.MeasurementTerminal{}

	if rows, e := db.Query(query, params.clone().add(limit, offset).values...); e != nil {
		return nil, 0, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			terminal := &model.MeasurementTerminal{}

			scanRows(db, rows, terminal, "t")

			records = append(records, terminal)
		})
	}

	if total, e := db.SelectInt(fmt.Sprintf(`SELECT COUNT(*) FROM measurement_terminal AS t %s`, where), params.values...); e != nil {
		return nil, 0, e
	} else {
		return records, total, nil
	}
}

// 計測端末情報を取得する。
func FetchTerminal(
	db model.QueryExecutor,
	id int,
) (*model.MeasurementTerminal, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM measurement_terminal AS t WHERE t.id = $1`,
		prefixColumns(model.MeasurementTerminal{}, "t", "t"),
	)

	if rows, e := db.Query(query, id); e != nil {
		return nil, e
	} else {
		var record *model.MeasurementTerminal

		safeRowsIterator(rows, func(rows *sql.Rows) {
			record = &model.MeasurementTerminal{}

			scanRows(db, rows, record, "t")
		})

		return record, nil
	}
}