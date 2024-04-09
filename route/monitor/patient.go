package monitor

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
// @tags [monitor] Patient
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param minutes query int false "期間長(分)。"
// @param end query string false "末尾日時。RFC3339形式。"
// @param limit query int false "最大取得件数。"
// @param offset query int false "取得オフセット。"
// @success 200 {object} listPatientsResponse "患者一覧。"
// @failure 400 {object} shared.ErrorResponse "バリデーションエラー。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/patients [get]
func listPatients(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

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

	results, total, err := service.List(me.Hospital.Id, begin, end, query.Limit, query.Offset)

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
// @tags [monitor] Patient
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param patient_id path int true "患者ID。"
// @success 200 {object} model.Patient "患者情報。"
// @failure 404 {object} shared.ErrorResponse "患者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/patients/{patient_id} [get]
func fetchPatient(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("patient_id")

	service := shared.CreateService(S.PatientService{}, c).(*S.PatientService)

	result, err := service.Fetch(id, me.Hospital.Id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

type updatePatientBody struct {
	Name              *string    `json:"name" maxLength:"64"`
	Age               *int       `json:"age"`
	NumChildren       *int       `json:"numChildren"`
	CesareanScar      *bool      `json:"cesareanScar"`
	DeliveryTime      *int       `json:"deliveryTime"`
	BloodLoss         *int       `json:"bloodLoss"`
	BirthWeight       *int       `json:"birthWeight"`
	BirthDatetime     *time.Time `json:"birthDatetime"`
	GestationalDays   *int       `json:"gestationalDays"`
	ApgarScore1Min    *int       `json:"apgarScore1min"`
	ApgarScore5Min    *int       `json:"apgarScore5min"`
	UmbilicalBlood    *int       `json:"umbilicalBlood"`
	EmergencyCesarean *bool      `json:"emergencyCesarean"`
	InstrumentalLabor *bool      `json:"instrumentalLabor"`
	Memo              string     `json:"memo" maxLength:"2000"`
}

// updatePatient godoc
// @summary 患者情報を更新する。
// @description 入力項目の内、存在しないもしくは`null`のものは更新されない。
// @tags [monitor] Patient
// @produce json
// @param Authorization header string true "Bearerトークン。"
// @param patient_id path int true "患者ID。"
// @param patient body updatePatientBody true "患者情報。"
// @success 200 {object} model.Patient "更新した患者情報。"
// @failure 404 {object} shared.ErrorResponse "患者が存在しない。"
// @failure 500 {object} shared.ErrorResponse "サーバエラーが発生。"
// @router /1/patients/{patient_id} [put]
func updatePatient(c *shared.Context) error {
	me := c.Get(shared.ContextMeKey).(*model.HospitalDoctor)

	id := c.IntParam("patient_id")

	body := &updatePatientBody{}

	if e := c.Bind(body); e != nil {
		return e
	}

	if e := (v.Errors{
		"name": v.Validate(body.Name, v.RuneLength(0, 64)),
		"age": v.Validate(body.Age, v.Min(0)),
		"numChildren": v.Validate(body.NumChildren, v.Min(0)),
		"deliveryTime": v.Validate(body.DeliveryTime, v.Min(0)),
		"bloodLoss": v.Validate(body.BloodLoss, v.Min(0)),
		"birthWeight": v.Validate(body.BirthWeight, v.Min(0)),
		"gestationalDays": v.Validate(body.GestationalDays, v.Min(0)),
		"umbilicalBlood": v.Validate(body.UmbilicalBlood, v.Min(0)),
		"memo": v.Validate(body.Memo, v.RuneLength(0, 2000)),
	}).Filter(); e != nil {
		return e
	}

	service := shared.CreateService(S.PatientTxService{}, c).(*S.PatientTxService)

	err := service.CheckUpdateByDoctor(me, id)

	if err != nil {
		return err
	}

	result, err := service.Update(
		id,
		body.Name,
		body.Age,
		body.NumChildren,
		body.CesareanScar,
		body.DeliveryTime,
		body.BloodLoss,
		body.BirthWeight,
		body.BirthDatetime,
		body.GestationalDays,
		body.ApgarScore1Min,
		body.ApgarScore5Min,
		body.UmbilicalBlood,
		body.EmergencyCesarean,
		body.InstrumentalLabor,
		body.Memo,
	)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}