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

type listHospitalsQuery struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

type listHospitalsResponse struct {
	Hospitals []*model.Hospital `json:"hospitals"`
	Total     int64             `json:"total"`
	Limit     int               `json:"limit"`
	Offset    int               `json:"offset"`
}

// listHospitals godoc
// @summary 病院一覧を取得する。
// @tags [admin] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param limit query int false "最大取得件数。"
// @param offset query int false "取得オフセット。"
// @success 200 {object} listHospitalsResponse "病院一覧。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals [get]
func listHospitals(c *shared.Context) error {
	query := &listHospitalsQuery{100, 0}

	if e := c.Bind(query); e != nil {
		return e
	}

	if e := (v.Errors{
		"limit": v.Validate(query.Limit, v.Min(0)),
		"offset": v.Validate(query.Offset, v.Min(0)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.HospitalService{}, c).(*S.HospitalService)

	results, total, err := service.List(query.Limit, query.Offset)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listHospitalsResponse{
		Hospitals: results,
		Total: total,
		Limit: query.Limit,
		Offset: query.Offset,
	})
}

// fetchHospital godoc
// @summary 病院情報を取得する。
// @tags [admin] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @success 200 {object} model.Hospital "病院情報。"
// @failure 404 {object} shared.ErrorResponse "病院が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id} [get]
func fetchHospital(c *shared.Context) error {
	id := c.IntParam("hospital_id")

	service := shared.CreateService(S.HospitalService{}, c).(*S.HospitalService)

	result, err := service.Fetch(id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type hospitalBody struct {
	Name string `json:"name" maxlength:"64"`
	Memo string `json:"memo" maxlength:"2000"`
}

// createHospital godoc
// @summary 病院を登録する。
// @tags [admin] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital body hospitalBody true "病院情報。"
// @success 201 {object} model.Hospital "登録した病院情報。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals [post]
func createHospital(c *shared.Context) error {
	body := &hospitalBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"name": v.Validate(body.Name, v.Required, v.RuneLength(0, 64)),
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.HospitalTxService{}, c).(*S.HospitalTxService)

	result, err := service.Create(body.Name, body.Memo)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, result)
}

// updateHospital godoc
// @summary 病院情報を更新する。
// @tags [admin] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param hospital body hospitalBody true "病院情報。"
// @success 200 {object} model.Hospital "更新した病院情報。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "病院が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id} [put]
func updateHospital(c *shared.Context) error {
	id := c.IntParam("hospital_id")

	body := &hospitalBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"name": v.Validate(body.Name, v.Required, v.RuneLength(0, 64)),
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.HospitalTxService{}, c).(*S.HospitalTxService)

	result, err := service.Update(id, body.Name, body.Memo)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// deleteHospital godoc
// @summary 病院を削除する。
// @tags [admin] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @success 204 "処理に成功。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id} [delete]
func deleteHospital(c *shared.Context) error {
	id := c.IntParam("hospital_id")

	service := shared.CreateService(S.HospitalTxService{}, c).(*S.HospitalTxService)

	err := service.Delete(id)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// generateApiKey godoc
// @summary CTG機器から利用するAPIキーを発行する。
// @tags [admin] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @success 201 {object} model.CTGAuthentication "発行したAPIキー。"
// @failure 404 {object} shared.ErrorResponse "病院が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /admin/hospitals/{hospital_id}/api_key [post]
func generateApiKey(c *shared.Context) error {
	id := c.IntParam("hospital_id")

	service := shared.CreateService(S.CTGTxService{}, c).(*S.CTGTxService)

	result, err := service.GenerateApiKey(id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, result)
}