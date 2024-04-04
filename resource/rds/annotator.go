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

// ログインIDから判定者を取得する。
// パスワードがnil以外の場合、パスワード認証を行う。
func FetchAnnotatorByLoginId(
	db model.QueryExecutor,
	loginId string,
	password *string,
) (*model.Annotator, error) {
	q := andQuery().add("a.login_id = $1", loginId)

	if password != nil {
		q.add("a.password = encode(digest($2, 'sha256'), 'hex')", password)
	}

	where, params := q.where()

	query := fmt.Sprintf(`SELECT * FROM annotator AS a %s`, where)

	result := model.Annotator{}

	if e := db.SelectOne(&result, query, params.values...); e != nil {
		if e == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, e
		}
	} else {
		return &result, nil
	}
}

func UpdateAnnotatorTokenVersion(
	db model.QueryExecutor,
	id int,
	version string,
	now time.Time,
) error {
	_, err := db.Exec(`UPDATE annotator SET token_version = $1, modified_at = $2 WHERE id = $3`, version, now, id)
	return err
}