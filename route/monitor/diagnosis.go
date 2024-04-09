package monitor

/*
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

type listDiagnosesResponse struct {
	Diagnoses []*model.DiagnosisEntity `json:"diagnoses"`
}

// listDiagnoses godoc
// @summary ある計測に関する全ての診断を取得する。
// @tags [monitor] Diagnosis
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @success 200 {object} listDiagnosesResponse "診断一覧。"
// @failure 404 {object} shared.ErrorResponse "計測記録が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/diagnoses [get]
func listDiagnoses(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	service := shared.CreateService(S.DiagnosisService{}, c).(*S.DiagnosisService)

	results, err := service.ListByMeasurement(measurementId)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listDiagnosesResponse{
		Diagnoses: results,
	})
}

// fetchDiagnosis godoc
// @summary 診断記録を取得する。
// @tags [monitor] Diagnosis
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param diagnosis_id path int true "診断ID。"
// @success 200 {object} model.DiagnosisEntity "診断情報。"
// @failure 404 {object} shared.ErrorResponse "診断が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/diagnoses/{diagnosis_id} [get]
func fetchDiagnosis(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("diagnosis_id")

	service := shared.CreateService(S.DiagnosisService{}, c).(*S.DiagnosisService)

	err := service.CheckAccessByDoctor(me, id, measurementId)

	if err != nil {
		return err
	}

	result, err := service.FetchByMeasurement(id, measurementId)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type diagnosisBody struct {
	Memo string `json:"memo"`
}

// updateDiagnosis godoc
// @summary 診断記録を更新する。
// @tags [monitor] Diagnosis
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param diagnosis_id path int true "診断ID。"
// @param diagnosis body diagnosisBody true "更新内容。"
// @success 200 {object} model.DiagnosisEntity "診断情報。"
// @failure 404 {object} shared.ErrorResponse "診断が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/diagnoses/{diagnosis_id} [put]
func updateDiagnosis(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("diagnosis_id")

	service := shared.CreateService(S.DiagnosisTxService{}, c).(*S.DiagnosisTxService)

	err := service.CheckUpdateByDoctor(me, id, measurementId)

	if err != nil {
		return err
	}

	body := &diagnosisBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	result, err := service.Update(id, body.Memo)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}
*/