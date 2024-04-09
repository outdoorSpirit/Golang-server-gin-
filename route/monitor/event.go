package monitor

import (
	//"fmt"
	"net/http"
	"time"

	v "github.com/go-ozzo/ozzo-validation/v4"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	//"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/route/view"
)

type listComputedEventsQuery struct {
	Minutes int `query:"minutes"`
	End     time.Time `query:"end"`
}

type listComputedEventsResponse struct {
	Events []*model.ComputedEventEntity `json:"events"`
}

// listComputedEvents godoc
// @summary ある計測に関する、期間内の自動診断イベントを新しい順に取得する。
// @tags [monitor] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param minutes query int false "期間長(分)。"
// @param end query string false "末尾日時。RFC3339形式。"
// @success 200 {object} listComputedEventsResponse "イベント一覧。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/events [get]
func listComputedEvents(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	query := &listComputedEventsQuery{}

	if e := c.Bind(query); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	begin := query.End.Add(time.Duration(-query.Minutes)*time.Minute)

	results, err := service.ListComputedEventsInRange(measurementId, begin, query.End, false)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listComputedEventsResponse{
		Events: results,
	})
}

// fetchComputedEvent godoc
// @summary 自動診断イベントを取得する。
// @tags [monitor] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param event_id path int true "イベントID。"
// @success 200 {object} model.ComputedEventEntity "イベント情報。"
// @failure 404 {object} shared.ErrorResponse "イベントが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/events/{event_id} [get]
func fetchComputedEvent(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("event_id")

	if e := checkComputedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	result, err := service.FetchComputedEvent(id, true)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type listAnnotatedEventsQuery struct {
	Minutes int `query:"minutes"`
	End     time.Time `query:"end"`
}

type listAnnotatedEventsResponse struct {
	Annotations []*model.AnnotatedEventEntity `json:"annotations"`
}

// listAnnotatedEvents godoc
// @summary ある計測に関するアノテーションを新しい順に取得する。
// @tags [monitor] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param minutes query int false "期間長(分)。"
// @param end query string false "末尾日時。RFC3339形式。"
// @success 200 {object} listAnnotatedEventsResponse "アノテーション一覧。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/annotations [get]
func listAnnotatedEvents(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	query := &listAnnotatedEventsQuery{}

	if e := c.Bind(query); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	begin := query.End.Add(time.Duration(-query.Minutes)*time.Minute)

	results, err := service.ListAnnotatedEventsInRange(measurementId, begin, query.End)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listAnnotatedEventsResponse{
		Annotations: results,
	})
}

// fetchAnnotatedEvent godoc
// @summary アノテーションを取得する。
// @tags [monitor] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param annotation_id path int true "アノテーションID。"
// @success 200 {object} model.AnnotatedEventEntity "イベント情報。"
// @failure 404 {object} shared.ErrorResponse "アノテーションが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/annotations/{annotation_id} [get]
func fetchAnnotatedEvent(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("annotation_id")

	if e := checkAnnotatedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	result, err := service.FetchAnnotatedEvent(id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type closeAnnotatedEventBody struct {
	Memo string `json:"memo"`
}

// closeAnnotatedEvent godoc
// @summary アノテーションを解決済みとする。
// @tags [monitor] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param annotation_id path int true "アノテーションID。"
// @param closing body closeAnnotatedEventBody true "完了関連情報。"
// @success 204 "処理に成功。"
// @failure 404 {object} shared.ErrorResponse "アノテーションが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/annotations/{annotation_id}/close [post]
func closeAnnotatedEvent(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("annotation_id")

	if e := checkAnnotatedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	service := shared.CreateService(S.EventTxService{}, c).(*S.EventTxService)

	body := &closeAnnotatedEventBody{}

	if e := c.Bind(body); e != nil {
		return e
	} else if e := (v.Errors{
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	err := service.CloseAnnotated(id, body.Memo)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}