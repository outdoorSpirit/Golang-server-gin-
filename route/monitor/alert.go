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

type collectAlertsQuery struct {
	Measurements []int `query:"measurements"`
}

type collectAlertsResponse struct {
	Annotations []*model.AnnotatedEventEntity `json:"annotations"`
}

// collectAlerts godoc
// @summary アラートのためのポーリングを受け付ける。
// @description 指定した計測それぞれに関する最新のアノテーションを取得する。
// @description レスポンスは、区間の古い順のアノテーションリスト。計測との対応は各アノテーションの`measurement`からとる。
// @tags [monitor] Alert
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurements query []int true "計測IDリスト。"
// @success 200 {object} collectAlertsResponse "最新アノテーションリスト。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "計測記録が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/alerts [get]
func collectAlerts(c *shared.Context) error {
	query := &collectAlertsQuery{}

	if e := c.Bind(query); e != nil {
		return e
	}

	if e := (v.Errors{
		"measurements": v.Validate(query.Measurements, v.Required),
	}).Filter(); e != nil {
		return e
	}

	for _, mid := range query.Measurements {
		if e := checkMeasurementAccess(c, mid); e != nil {
			return e
		}
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	annotations, err := service.ListAlertEvents(query.Measurements)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &collectAlertsResponse{annotations})
}