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

type listTerminalsQuery struct {
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

type listTerminalsResponse struct {
	Terminals []*model.MeasurementTerminal `json:"terminals"`
	Total     int64                        `json:"total"`
	Limit     int                          `json:"limit"`
	Offset    int                          `json:"offset"`
}

// listTerminals godoc
// @summary 計測端末一覧を取得する。
// @tags [monitor] Terminal
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param limit query int false "最大取得件数。"
// @param offset query int false "取得オフセット。"
// @success 200 {object} listTerminalsResponse "計測端末一覧。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/terminals [get]
func listTerminals(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	query := &listTerminalsQuery{100, 0}

	if e := c.Bind(query); e != nil {
		return e
	}

	if e := (v.Errors{
		"limit": v.Validate(query.Limit, v.Min(0)),
		"offset": v.Validate(query.Offset, v.Min(0)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.InstrumentService{}, c).(*S.InstrumentService)

	results, total, err := service.List(me.Hospital.Id, query.Limit, query.Offset)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listTerminalsResponse{
		Terminals: results,
		Total: total,
		Limit: query.Limit,
		Offset: query.Offset,
	})
}

// fetchTerminal godoc
// @summary 計測端末情報を取得する。
// @tags [monitor] Terminal
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param terminal_id path int true "計測端末ID。"
// @success 200 {object} model.MeasurementTerminal "計測端末情報。"
// @failure 404 {object} shared.ErrorResponse "計測端末が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/terminals/{terminal_id} [get]
func fetchTerminal(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("terminal_id")

	service := shared.CreateService(S.InstrumentService{}, c).(*S.InstrumentService)

	result, err := service.Fetch(id, me.Hospital.Id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type updateTerminalBody struct {
	Memo string `json:"memo" maxLength:"2000"`
}

// updateTerminal godoc
// @summary 計測端末情報を更新する。
// @tags [monitor] Terminal
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param terminal body updateTerminalBody true "更新内容。"
// @success 200 {object} model.MeasurementTerminal "計測端末情報。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測端末が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/terminals/{terminal_id} [put]
func updateTerminal(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("terminal_id")

	body := &updateTerminalBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.InstrumentTxService{}, c).(*S.InstrumentTxService)

	err := service.CheckUpdateByDoctor(me, id)

	if err != nil {
		return err
	}

	result, err := service.Update(id, body.Memo)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}