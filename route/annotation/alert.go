package annotation

import (
	"net/http"
	//"time"

	//v "github.com/go-ozzo/ozzo-validation/v4"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	//"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/route/view"
)

type collectUnreadAlertQuery struct {
	LatestEvent *int `query:"latest"`
}

type collectUnreadAlertResponse struct {
	Events []*model.ComputedEventEntity `json:"events"`
}

// collectUnreadAlerts godoc
// @summary アラートのためのポーリングを受け付ける。
// @description 一定以上のリスクかつ、5分以内にデータのある計測に関する未読イベントを古い順に取得する。
// @description 一つの計測に複数の対象イベントが存在する場合、個別のイベントとして全て返される。
// @tags [annotation] Alert
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param hospital_uuid path string true "病院UUID。"
// @param latest query int false "既読の最終イベントID。省略時は既定の時間分遡った先頭から。"
// @success 200 {object} collectUnreadAlertResponse "未読イベントリスト。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "病院が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /annotation/hospitals/{hospital_uuid}/alerts [get]
func collectUnreadAlerts(c *shared.Context) error {
	hospital, err := checkHospitalAccess(c)

	if err != nil {
		return err
	}

	query := &collectUnreadAlertQuery{}

	if e := c.Bind(query); e != nil {
		return e
	}

	service := shared.CreateService(S.EventService{}, c).(*S.EventService)

	results, err := service.ListUnreadComputedEvents(hospital.Id, query.LatestEvent)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &collectUnreadAlertResponse{results})
}