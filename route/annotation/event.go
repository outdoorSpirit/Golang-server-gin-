package annotation

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
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param minutes query int false "期間長(分)。"
// @param end query string false "末尾日時。RFC3339形式。"
// @success 200 {object} listComputedEventsResponse "イベント一覧。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/events [get]
func listComputedEvents(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId, nil); e != nil {
		return e
	}

	query := &listComputedEventsQuery{}

	if e := c.Bind(query); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	begin := query.End.Add(time.Duration(-query.Minutes)*time.Minute)

	results, err := service.ListComputedEventsInRange(measurementId, begin, query.End, true)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listComputedEventsResponse{
		Events: results,
	})
}

// fetchComputedEvent godoc
// @summary 自動診断イベントを取得する。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param event_id path int true "イベントID。"
// @success 200 {object} model.ComputedEventEntity "イベント情報。"
// @failure 404 {object} shared.ErrorResponse "イベントが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/events/{event_id} [get]
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

// showComputedEvent godoc
// @summary 自動診断イベントを病院側で表示可能とする。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param event_id path int true "イベントID。"
// @success 204 "処理に成功。"
// @failure 404 {object} shared.ErrorResponse "イベントが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/events/{event_id}/show [post]
func showComputedEvent(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("event_id")

	if e := checkComputedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	service := shared.CreateService(S.EventTxService{}, c).(*S.EventTxService)

	err := service.ToggleComputedEventVisibility(id, true)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// hideComputedEvent godoc
// @summary 自動診断イベントを病院側で非表示とする。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param event_id path int true "イベントID。"
// @success 204 "処理に成功。"
// @failure 404 {object} shared.ErrorResponse "イベントが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/events/{event_id}/show [delete]
func hideComputedEvent(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("event_id")

	if e := checkComputedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	service := shared.CreateService(S.EventTxService{}, c).(*S.EventTxService)

	err := service.ToggleComputedEventVisibility(id, false)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// suspendComputedEvent godoc
// @summary 自動診断イベントを待機中とする。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param event_id path int true "イベントID。"
// @success 204 "処理に成功。"
// @failure 404 {object} shared.ErrorResponse "イベントが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/events/{event_id}/suspend [post]
func suspendComputedEvent(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("event_id")

	if e := checkComputedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	service := shared.CreateService(S.EventTxService{}, c).(*S.EventTxService)

	err := service.SuspendComputedEvent(id)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

type listAnnotatedEventsQuery struct {
	Minutes int `query:"minutes"`
	End     time.Time `query:"end"`
}

type listAnnotatedEventsResponse struct {
	Annotations []*model.AnnotatedEventEntity `json:"annotations"`
}

// listAnnotatedEvents godoc
// @summary ある計測に関する、期間内のアノテーションを新しい順に取得する。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param minutes query int false "期間長(分)。"
// @param end query string false "末尾日時。RFC3339形式。"
// @success 200 {object} listAnnotatedEventsResponse "アノテーション一覧。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/annotations [get]
func listAnnotatedEvents(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId, nil); e != nil {
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
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param annotation_id path int true "アノテーションID。"
// @success 200 {object} model.AnnotatedEventEntity "アノテーション情報。"
// @failure 404 {object} shared.ErrorResponse "アノテーションが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/annotations/{annotation_id} [get]
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

type registerAnnotatedEventBody struct {
	EventId    *int      `json:"eventId"`
	Risk       int       `json:"risk"`
	Memo       string    `json:"memo" maxlength:"2000"`
	RangeFrom  time.Time `json:"rangeFrom"`
	RangeUntil time.Time `json:"rangeUntil"`
}

// registerAnnotatedEvent godoc
// @summary ある計測に対してアノテーションを登録する。
// @description `eventId`をセットすることで、そのイベントにリンクしたアノテーションとなる。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param annotation body registerAnnotatedEventBody true "アノテーション情報。"
// @success 201 {object} model.AnnotatedEvent "登録したアノテーション情報。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/annotations [post]
func registerAnnotatedEvent(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.Annotator)

	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId, nil); e != nil {
		return e
	}

	body := &registerAnnotatedEventBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"risk": v.Validate(body.Risk, v.In(1, 2, 3, 4, 5)),
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.EventTxService{}, c).(*S.EventTxService)

	result, err := service.RegisterAnnotated(measurementId, me.Id, body.EventId, body.Risk, body.Memo, body.RangeFrom, body.RangeUntil)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, result)
}

type updateAnnotatedEventBody struct {
	Risk int    `json:"risk"`
	Memo string `json:"memo" maxlength:"2000"`
}

// updateAnnotatedEvent godoc
// @summary アノテーションを更新する。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param annotation_id path int true "アノテーションID。"
// @param annotation body updateAnnotatedEventBody true "アノテーション情報。"
// @success 204 "処理に成功。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "アノテーションが存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/annotations/{annotation_id} [put]
func updateAnnotatedEvent(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.Annotator)

	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("annotation_id")

	if e := checkAnnotatedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	body := &updateAnnotatedEventBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"risk": v.Validate(body.Risk, v.In(1, 2, 3, 4, 5)),
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.EventTxService{}, c).(*S.EventTxService)

	err := service.UpdateAnnotated(id, me.Id, body.Risk, body.Memo)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// deleteAnnotatedEvent godoc
// @summary アノテーションを削除する。
// @tags [annotation] Event
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param measurement_id path int true "計測記録ID。"
// @param annotation_id path int true "アノテーションID。"
// @success 204 "処理に成功。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/measurements/{measurement_id}/annotations/{annotation_id} [delete]
func deleteAnnotatedEvent(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")
	id := c.IntParam("annotation_id")

	if e := checkAnnotatedEventAccess(c, measurementId, id); e != nil {
		return e
	}

	service := shared.CreateService(S.EventTxService{}, c).(*S.EventTxService)

	err := service.DeleteAnnotated(id)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}