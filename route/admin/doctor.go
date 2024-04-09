package admin

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

type listDoctorsQuery struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

type listDoctorsResponse struct {
	Doctors []*model.Doctor `json:"doctors"`
	Total   int64           `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}

// listDoctors godoc
// @summary 病院の医者一覧を取得する。
// @tags [admin] Doctor
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param limit query int false "最大取得件数。"
// @param offset query int false "取得オフセット。"
// @success 200 {object} listDoctorsResponse "医者一覧。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id}/doctors [get]
func listDoctors(c *shared.Context) error {
	hospitalId := c.IntParam("hospital_id")

	query := &listDoctorsQuery{100, 0}

	if e := c.Bind(query); e != nil {
		return e
	}

	if e := (v.Errors{
		"limit": v.Validate(query.Limit, v.Min(0)),
		"offset": v.Validate(query.Offset, v.Min(0)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.DoctorService{}, c).(*S.DoctorService)

	results, total, err := service.List(hospitalId, query.Limit, query.Offset)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listDoctorsResponse{
		Doctors: results,
		Total: total,
		Limit: query.Limit,
		Offset: query.Offset,
	})
}

// fetchDoctor godoc
// @summary 医者情報を取得する。
// @tags [admin] Doctor
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param doctor_id path int true "医者ID。"
// @success 200 {object} model.Doctor "医者情報。"
// @failure 404 {object} shared.ErrorResponse "医者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id}/doctors/{doctor_id} [get]
func fetchDoctor(c *shared.Context) error {
	id, err := checkDoctorInHospital(c)

	if err != nil {
		return err
	}

	service := shared.CreateService(S.DoctorService{}, c).(*S.DoctorService)

	result, err := service.Fetch(id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type doctorBody struct {
	LoginId  string `json:"loginId" maxlength:"32"`
	Password string `json:"password" maxlength:"32"`
	Name     string `json:"name" maxlength:"64"`
}

// createDoctor godoc
// @summary 医者を登録する。
// @tags [admin] Doctor
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param doctor body doctorBody true "医者情報。"
// @success 201 {object} model.Doctor "登録した医者情報。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。ログインIDが既存。"
// @failure 404 {object} shared.ErrorResponse "病院が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id}/doctors [post]
func createDoctor(c *shared.Context) error {
	hospitalId := c.IntParam("hospital_id")

	body := &doctorBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"loginId": v.Validate(body.LoginId, v.Required, v.RuneLength(0, 32)),
		"password": v.Validate(body.Password, v.Required, v.RuneLength(0, 32)),
		"name": v.Validate(body.Name, v.Required, v.RuneLength(0, 64)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.DoctorTxService{}, c).(*S.DoctorTxService)

	result, err := service.Create(hospitalId, body.LoginId, body.Password, body.Name)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, result)
}

type updateBody struct {
	Name string `json:"name" maxlength:"64"`
}

// updateDoctor godoc
// @summary 医者情報を更新する。
// @tags [admin] Doctor
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param doctor_id path int true "医者ID。"
// @param doctor body updateBody true "医者情報。"
// @success 200 {object} model.Doctor "更新した医者情報。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "医者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id}/doctors/{doctor_id} [put]
func updateDoctor(c *shared.Context) error {
	id, err := checkDoctorInHospital(c)

	if err != nil {
		return err
	}

	body := &updateBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"name": v.Validate(body.Name, v.Required, v.RuneLength(0, 64)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.DoctorTxService{}, c).(*S.DoctorTxService)

	result, err := service.Update(id, body.Name)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type updatePasswordBody struct {
	Password string `json:"password" maxlength:"32"`
}

// updateDoctorPassword godoc
// @summary 医者のパスワードを更新する。
// @tags [admin] Doctor
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param doctor_id path int true "医者ID。"
// @param doctor body updatePasswordBody true "更新パスワード。"
// @success 204 "処理に成功。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "医者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id}/doctors/{doctor_id}/password [put]
func updateDoctorPassword(c *shared.Context) error {
	id, err := checkDoctorInHospital(c)

	if err != nil {
		return err
	}

	body := &updatePasswordBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"password": v.Validate(body.Password, v.Required, v.RuneLength(0, 32)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.DoctorTxService{}, c).(*S.DoctorTxService)

	err = service.UpdatePassword(id, body.Password)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// deleteDoctor godoc
// @summary 医者を削除する。
// @tags [admin] Doctor
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param doctor_id path int true "医者ID。"
// @success 204 "処理に成功。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id}/doctors/{doctor_id} [delete]
func deleteDoctor(c *shared.Context) error {
	id, err := checkDoctorInHospital(c)

	if err != nil {
		return err
	}

	service := shared.CreateService(S.DoctorTxService{}, c).(*S.DoctorTxService)

	err = service.Delete(id)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func checkDoctorInHospital(c *shared.Context) (int, error) {
	id := c.IntParam("doctor_id")
	hospitalId := c.IntParam("hospital_id")

	service := shared.CreateService(S.DoctorService{}, c).(*S.DoctorService)

	err := service.CheckDoctorInHospital(id, hospitalId)

	if err != nil {
		return 0, err
	} else {
		return id, nil
	}
}