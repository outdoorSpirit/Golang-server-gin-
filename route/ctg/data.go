package ctg

import (
	"net/http"
	//"time"

	//v "github.com/go-ozzo/ozzo-validation/v4"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	//"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/route/view"
)

type uploadResponse struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
}

// uploadCTG godoc
// @summary CTGデータをjson配列形式でアップロードする。
// @tags [ctg] Data
// @accept json
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param data body []S.CTGData true "json配列形式のCTGデータ。"
// @success 201 {object} uploadResponse "登録したレコード数とエラー数。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /ctg/data [post]
func uploadCTG(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.CTGAuthentication)

	body := []S.CTGData{}

	if e := c.Bind(&body); e != nil {
		return e
	}

	service := shared.CreateService(S.DataTxService{}, c).(*S.DataTxService)

	stats, err := service.RegisterCTGData(me.HospitalId, body)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, &uploadResponse{
		Success: stats.SuccessFHR1,
		Failure: len(stats.FailureFHR1),
	})
}