package annotation

/*
 * イベントAPIに移行。
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
// @tags [annotation] Diagnosis
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @success 200 {object} listDiagnosesResponse "診断一覧。"
// @failure 404 {object} shared.ErrorResponse "計測記録が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/diagnoses [get]
func listDiagnoses(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId, nil); e != nil {
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
// @tags [annotation] Diagnosis
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param diagnosis_id path int true "診断ID。"
// @success 200 {object} model.DiagnosisEntity "診断情報。"
// @failure 404 {object} shared.ErrorResponse "診断が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/diagnoses/{diagnosis_id} [get]
func fetchDiagnosis(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	var hospital model.Hospital

	if e := checkMeasurementAccess(c, measurementId, &hospital); e != nil {
		return e
	}

	id := c.IntParam("diagnosis_id")

	service := shared.CreateService(S.DiagnosisService{}, c).(*S.DiagnosisService)

	err := service.CheckAccessByHospital(hospital.Id, id, measurementId)

	if err != nil {
		return err
	}

	result, err := service.FetchByMeasurement(id, measurementId)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// TODO 自動診断の非表示

type diagnosisBody struct {
	Contents []S.DiagnosisContentItem `json:"contents"`
	Memo     string                   `json:"memo"`
}

// registerDiagnosis godoc
// @summary 診断を登録する。
// @tags [annotation] Diagnosis
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_id path int true "病院ID。"
// @param measurement_id path int true "計測ID。"
// @param diagnosis body diagnosisBody true "診断情報。"
// @success 201 {object} model.DiagnosisEntity "登録した診断。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/diagnoses [post]
func registerDiagnosis(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.Annotator)

	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId, nil); e != nil {
		return e
	}

	body := &diagnosisBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"contents": v.Validate(body.Contents, v.Required),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.DiagnosisTxService{}, c).(*S.DiagnosisTxService)

	result, err := service.RegisterByAnnotator(measurementId, me.Id, body.Memo, body.Contents)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, result)
}
*/