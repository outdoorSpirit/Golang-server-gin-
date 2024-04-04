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

// 病院をID順に取得する。
func ListHospitals(
	db model.QueryExecutor,
	limit int,
	offset int,
) ([]*model.Hospital, int64, error) {
	records := []*model.Hospital{}

	query := fmt.Sprintf(`SELECT * FROM hospital ORDER BY id ASC LIMIT $1 OFFSET $2`)

	if _, e := db.Select(&records, query, limit, offset); e != nil {
		return nil, 0, e
	}

	if total, e := db.SelectInt(`SELECT COUNT(*) FROM hospital`); e != nil {
		return nil, 0, e
	} else {
		return records, total, nil
	}
}

// 病院をIDから取得する。
func FetchHospital(
	db model.QueryExecutor,
	id int,
) (*model.Hospital, error) {
	record := model.Hospital{}

	query := fmt.Sprintf(`SELECT * FROM hospital WHERE id = $1`)

	if e := db.SelectOne(&record, query, id); e != nil {
		if e == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, e
		}
	} else {
		return &record, nil
	}
}

// UUIDから病院を取得する。
func InquireHospitalByUuid(
	db model.QueryExecutor,
	uuid string,
) (*model.Hospital, error) {
	record := model.Hospital{}

	query := fmt.Sprintf(`SELECT * FROM hospital WHERE uuid = $1`)

	if e := db.SelectOne(&record, query, uuid); e != nil {
		if e == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, e
		}
	} else {
		return &record, nil
	}
}