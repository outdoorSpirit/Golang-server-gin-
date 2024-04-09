package monitor

import (
	"net/http"
	//"time"

	v "github.com/go-ozzo/ozzo-validation/v4"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	//"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/route/view"
)

type loginBody struct {
	LoginId  string `json:"loginId"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
}

// login godoc
// @summary パスワード認証を行い、トークンを取得する。
// @tags [monitor] Login
// @produce json
// @param login body loginBody true "ログイン情報。"
// @success 200 {object} loginResponse "アクセストークン。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/login [post]
func login(c *shared.Context) error {
	body := &loginBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"loginId":  v.Validate(body.LoginId, v.Required),
		"password": v.Validate(body.Password, v.Required),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.DoctorService{}, c).(*S.DoctorService)

	me, err := service.Login(body.LoginId, body.Password)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &loginResponse{
		AccessToken: shared.CreateTokenWithStandardClaims(me.Doctor.LoginId, me.Doctor.TokenVersion),
	})
}

// logout godoc
// @summary ログアウトして、アクセストークンを無効化する。
// @tags [monitor] Login
// @produce json
// @param Authorization header string true "`Bearerトークン。"
// @param login body loginBody true "ログイン情報。"
// @success 204 "処理に成功。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/login [delete]
func logout(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	service := shared.CreateService(S.DoctorTxService{}, c).(*S.DoctorTxService)

	err := service.UpdateVersion(me.Doctor.Id)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}