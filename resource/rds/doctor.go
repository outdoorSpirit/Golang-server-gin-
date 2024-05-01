package rds

import (
	"fmt"
	"database/sql"
	//"encoding/json"
	//"fmt"
	//"strings"
	"time"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
)

// ログインIDから医師を取得する。
// パスワードがnil以外の場合、パスワード認証を行う。
func FetchDoctorByLoginId(
	db model.QueryExecutor,
	loginId string,
	password *string,
) (*model.HospitalDoctor, error) {
	q := andQuery().add("d.login_id = $1", loginId)

	if password != nil {
		q.add("d.password = encode(digest($2, 'sha256'), 'hex')", password)
	}

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s
		FROM
			doctor AS d
			INNER JOIN hospital AS h ON d.hospital_id = h.id
		%s`,
		prefixColumns(model.Doctor{}, "d", "d"),
		prefixColumns(model.Hospital{}, "h", "h"),
		where,
	)

	if rows, e := db.Query(query, params.values...); e != nil {
		return nil, e
	} else {
		var record *model.HospitalDoctor = nil

		safeRowsIterator(rows, func(rows *sql.Rows) {
			record = &model.HospitalDoctor{&model.Doctor{}, &model.Hospital{}}

			scanRows(db, rows, record.Doctor, "d")
			scanRows(db, rows, record.Hospital, "h")
		})

		return record, nil
	}
}

func UpdateDoctorTokenVersion(
	db model.QueryExecutor,
	id int,
	version string,
	now time.Time,
) error {
	_, err := db.Exec(`UPDATE doctor SET token_version = $1, modified_at = $2 WHERE id = $3`, version, now, id)
	return err
}

// 病院内の医者をID順に取得する。
func ListDoctorsInHospital(
	db model.QueryExecutor,
	hospitalId int,
	limit int,
	offset int,
) ([]*model.Doctor, int64, error) {
	query := `SELECT * FROM doctor WHERE hospital_id = $1 ORDER BY id ASC LIMIT $2 OFFSET $3`

	records := []*model.Doctor{}

	if _, e := db.Select(&records, query, hospitalId, limit, offset); e != nil {
		return nil, 0, e
	}

	if total, e := db.SelectInt(`SELECT COUNT(*) FROM doctor WHERE hospital_id = $1`, hospitalId); e != nil {
		return nil, 0, e
	} else {
		return records, total, nil
	}
}

// 病院と医者の関連を調べる。
func CheckDoctorInHospital(
	db model.QueryExecutor,
	id int,
	hospitalId int,
) (bool, error) {
	if r, e := FetchDoctor(db, id); e != nil {
		return false, e
	} else {
		return (r != nil && r.HospitalId == hospitalId), nil
	}
}

// 医者情報を取得する。
func FetchDoctor(
	db model.QueryExecutor,
	id int,
) (*model.Doctor, error) {
	record := model.Doctor{}

	if e := db.SelectOne(&record, `SELECT * FROM doctor WHERE id = $1`, id); e != nil {
		if e == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, e
		}
	} else {
		return &record, nil
	}
}

// 医者のパスワードハッシュを設定する。
func SetDoctorPassword(
	db model.QueryExecutor,
	id int,
	password string,
) error {
	if _, e := db.Exec(`UPDATE doctor SET password = encode(digest($1, 'sha256'), 'hex') WHERE id = $2`, password, id); e != nil {
		return e
	} else {
		return nil
	}
}