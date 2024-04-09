package annotation

import (
	"net/http"
	"time"

	v "github.com/go-ozzo/ozzo-validation/v4"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	//"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/route/view"
)

type listPatientsQuery struct {
	Minutes *int       `query:"minutes"`
	End     *time.Time `query:"end"`
	Limit   int `query:"limit"`
	Offset  int `query:"offset"`
}

type listPatientsResponse struct {
	Patients []*model.Patient `json:"patients"`
	Total    int64            `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// listPatients godoc
// @summary 患者一覧を取得する。
// @description `minutes` `end` が指定されている場合、その期間に計測のあった患者に絞り込む。片方のみの指定はバリデーションエラー。
// @tags [annotation] Patient
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param minutes query int false "期間長(分)。"
// @param end query string false "末尾日時。RFC3339形式。"
// @param limit query int false "最大取得件数。"
// @param offset query int false "取得オフセット。"
// @success 200 {object} listPatientsResponse "患者一覧。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/patients [get]
func listPatients(c *shared.Context) error {
	hospital, err := checkHospitalAccess(c)

	if err != nil {
		return err
	}

	query := &listPatientsQuery{nil, nil, 100, 0}

	if e := c.Bind(query); e != nil {
		return e
	}

	if e := (v.Errors{
		"limit": v.Validate(query.Limit, v.Min(0)),
		"offset": v.Validate(query.Offset, v.Min(0)),
	}).Filter(); e != nil {
		return e
	}

	var begin *time.Time = nil
	var end *time.Time = nil

	if query.Minutes != nil && query.End != nil {
		end = query.End
		b := end.Add(-time.Duration(*query.Minutes)*time.Minute)
		begin = &b
	} else if query.Minutes != nil || query.End != nil {
		return C.NewBadRequestError(
			"incomplete_time_range",
			"It is not allowed to set only one of end and minutes",
			map[string]interface{}{},
		)
	}

	service := shared.CreateService(S.PatientService{}, c).(*S.PatientService)

	results, total, err := service.List(hospital.Id, begin, end, query.Limit, query.Offset)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listPatientsResponse{
		Patients: results,
		Total: total,
		Limit: query.Limit,
		Offset: query.Offset,
	})
}

// fetchPatient godoc
// @summary 患者情報を取得する。
// @tags [annotation] Patient
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param patient_id path int true "患者ID。"
// @success 200 {object} model.Patient "患者情報。"
// @failure 404 {object} shared.ErrorResponse "患者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/patients/{patient_id} [get]
func fetchPatient(c *shared.Context) error {
	hospital, err := checkHospitalAccess(c)

	if err != nil {
		return err
	}

	id := c.IntParam("patient_id")

	service := shared.CreateService(S.PatientService{}, c).(*S.PatientService)

	if e := service.CheckAccessByHospital(hospital.Id, id); e != nil {
		return e
	}

	result, err := service.Fetch(id, hospital.Id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}