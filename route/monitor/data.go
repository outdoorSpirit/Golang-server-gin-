package monitor

import (
	"net/http"
	"time"

	//v "github.com/go-ozzo/ozzo-validation/v4"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
	//"github.com/spiker/spiker-server/resource/rds"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/route/view"
)

type listHeartRateQuery struct {
	Minutes *int       `query:"minutes"`
	End     *time.Time `query:"end"`
}

type listHeartRateResponse struct {
	Records []*model.SensorValue `json:"records"`
}

// listHeartRate godoc
// @summary 指定期間の心拍を古い順に取得する。
// @description 返されるデータは左閉右開。
// @tags [monitor] Data
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param minutes query int false "期間長(分)。省略時は先頭から。"
// @param end query string false "末尾日時。RFC3339形式。省略時は現在日時。"
// @success 200 {object} listHeartRateResponse "心拍記録。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "患者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/heartrates [get]
func listHeartRate(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	query := &listHeartRateQuery{}

	if e := c.Bind(query); e != nil {
		return e
	}

	service := shared.CreateService(S.DataService{}, c).(*S.DataService)

	begin, end := determineDataRange(query.Minutes, query.End)

	results, err := service.ListCTGData(measurementId, C.MeasurementTypeHeartRate, begin, end)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listHeartRateResponse{results})
}

type listTOCOQuery struct {
	Minutes *int       `query:"minutes"`
	End     *time.Time `query:"end"`
}

type listTOCOResponse struct {
	Records []*model.SensorValue `json:"records"`
}

// listTOC godoc
// @summary 指定期間のTOCOを古い順に取得する。
// @description 返されるデータは左閉右開。
// @tags [monitor] Data
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param measurement_id path int true "計測記録ID。"
// @param minutes query int false "期間長(分)。省略時は先頭から。"
// @param end query string false "末尾日時。RFC3339形式。省略時は現在日時。"
// @success 200 {object} listTOCOResponse "TOCO記録。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 404 {object} shared.ErrorResponse "患者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/measurements/{measurement_id}/tocos [get]
func listTOCO(c *shared.Context) error {
	measurementId := c.IntParam("measurement_id")

	if e := checkMeasurementAccess(c, measurementId); e != nil {
		return e
	}

	query := &listTOCOQuery{}

	if e := c.Bind(query); e != nil {
		return e
	}

	service := shared.CreateService(S.DataService{}, c).(*S.DataService)

	begin, end := determineDataRange(query.Minutes, query.End)

	results, err := service.ListCTGData(measurementId, C.MeasurementTypeTOCO, begin, end)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &listTOCOResponse{results})
}

func determineDataRange(minutes *int, end *time.Time) (*time.Time, *time.Time) {
	if end == nil {
		now := time.Now()
		end = &now
	}

	var begin *time.Time = nil

	if minutes != nil {
		b := end.Add(-time.Duration(*minutes)*time.Minute)
		begin = &b
	}

	return begin, end
}