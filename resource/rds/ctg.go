package rds

import (
	"database/sql"
	//"encoding/json"
	//"fmt"
	//"strings"
	//"time"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
)

// ログインIDから医師を取得する。
func FetchCTGByApiKey(
	db model.QueryExecutor,
	apiKey string,
) (*model.CTGAuthentication, error) {
	query := `SELECT * FROM ctg_authentication WHERE api_key = :api_key`

	record := model.CTGAuthentication{}

	if e := db.SelectOne(&record, query, map[string]interface{}{
		"api_key": apiKey,
	}); e != nil {
		return nil, e
	} else if e == sql.ErrNoRows {
		return nil, nil
	} else {
		return &record, nil
	}
}