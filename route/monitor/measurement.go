package monitor

import (
	"net/http"
	//"time"

	v "github.com/go-ozzo/ozzo-validation/v4"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	//"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/route/view"
)

type listMeasurementsQuery struct {
	Limit    int `query:"limit"`
	Offset   int `query:"offset"`
	Patient  *int `query:"patient"`
	Terminal *int `query:"terminal"`
}

type listMeasurementsResponse struct {
	Measurements []*model.MeasurementEntity `json:"measurements"`
	Total        int64                      `json:"total"`
	Limit        int                        `json:"limit"`
	Offset       int                        `json:"offset"`
}

// listMeasurements godoc
// @summary 院内で行われた計測記録一覧を新しい順に取得する。
// @description `terminal`と`patient`は排他。両方指定した場合はエラー。いずれも指定しない場合、全ての計測記録を取得する。
// @tags [monitor] Measurement
// @produce json
// @param Authorization header string true "`Bearerトークン。"
// @param limit query int false "最大取得件数。"
// @param offset query int false "取得オフセット。"
// @param patient query int false "患者ID。`terminal`と排他。"
// @param terminal query int false "機器端末ID。`patient`と排他。"
// @success 200 {object} listMeasurementsResponse "計測記録一覧。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements [get]
func listMeasurements(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	query := &listMeasurementsQuery{100, 0, nil, nil}

	if e := c.Bind(query); e != nil {
		return e
	}

	if e := (v.Errors{
		"limit": v.Validate(query.Limit, v.Min(0)),
		"offset": v.Validate(query.Offset, v.Min(0)),
	}).Filter(); e != nil {
		return e
	}

	if query.Patient != nil && query.Terminal != nil {
		return C.NewBadRequestError(
			"patient_or_terminal",
			"Just one of patient ID or terminal ID can be set to query",	
			map[string]interface{}{},
		)
	}

	service := shared.CreateService(S.MeasurementService{}, c).(*S.MeasurementService)

	results, total, err := service.List(me.Hospital.Id, query.Patient, query.Terminal, query.Limit, query.Offset)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listMeasurementsResponse{
		Measurements: results,
		Total:        total,
		Limit:        query.Limit,
		Offset:       query.Offset,
	})
}

// fetchMeasurement godoc
// @summary 計測記録情報を取得する。
// @tags [monitor] Measurement
// @produce json
// @param Authorization header string true "`Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @success 200 {object} model.MeasurementEntity "計測記録情報。"
// @failure 404 {object} shared.ErrorResponse "計測記録が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id} [get]
func fetchMeasurement(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("measurement_id")

	service := shared.CreateService(S.MeasurementService{}, c).(*S.MeasurementService)

	err := service.CheckAccessByDoctor(me, id)

	if err != nil {
		return err
	}

	result, err := service.Fetch(id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type getSilentStateResponse struct {
	IsSilent   bool                    `json:"isSilent"`
	Parameters *model.MeasurementAlert `json:"parameters"`
}

// getSilentState godoc
// @summary 計測のサイレント状態を取得する。
// @tags [monitor] Measurement
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @success 200 {object} getSilentStateResponse "サイレント状態。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/silent [get]
func getSilentState(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("measurement_id")

	service := shared.CreateService(S.MeasurementService{}, c).(*S.MeasurementService)

	err := service.CheckAccessByDoctor(me, id)

	if err != nil {
		return err
	}

	result, err := service.GetSilent(id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &getSilentStateResponse{result != nil, result})
}

// setSilentState godoc
// @summary 計測をサイレント状態にする。
// @tags [monitor] Measurement
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @success 201 {object} model.MeasurementAlert "登録したサイレント状態。"
// @failure 400 {object} shared.ErrorResponse "既にサイレント状態。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/silent [post]
func setSilentState(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("measurement_id")

	service := shared.CreateService(S.MeasurementTxService{}, c).(*S.MeasurementTxService)

	err := service.CheckUpdateByDoctor(me, id)

	if err != nil {
		return err
	}

	result, err := service.SetSilent(id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, result)
}

type closeMeasurementBody struct {
	Memo string `json:"memo"`
}

// closeMeasurement godoc
// @summary 計測を完了状態にする。
// @tags [monitor] Measurement
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param closing body closeMeasurementBody true "完了関連情報。"
// @success 204 "処理に成功。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/close [post]
func closeMeasurement(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("measurement_id")

	service := shared.CreateService(S.MeasurementTxService{}, c).(*S.MeasurementTxService)

	if e := service.CheckUpdateByDoctor(me, id); e != nil {
		return e
	}

	body := &closeMeasurementBody{}

	if e := c.Bind(body); e != nil {
		return e
	} else if e := (v.Errors{
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	if e := service.Close(id, body.Memo); e != nil {
		return e
	}

	return c.NoContent(http.StatusNoContent)
}