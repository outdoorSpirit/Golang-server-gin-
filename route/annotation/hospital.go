package annotation

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
// @tags [annotation] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param limit query int false "最大取得件数。"
// @param offset query int false "取得オフセット。"
// @success 200 {object} listHospitalsResponse "病院一覧。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals [get]
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
// @tags [annotation] Hospital
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @success 200 {object} model.Hospital "病院情報。"
// @failure 404 {object} shared.ErrorResponse "病院が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid} [get]
func fetchHospital(c *shared.Context) error {
	hospital, err := checkHospitalAccess(c)

	if err != nil {
		return err
	}

	service := shared.CreateService(S.HospitalService{}, c).(*S.HospitalService)

	result, err := service.Fetch(hospital.Id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}