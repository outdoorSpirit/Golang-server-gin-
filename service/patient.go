package service

import (
	"fmt"
	"time"

	"gopkg.in/gorp.v2"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
)

type PatientService struct {
	*Service
	DB *gorp.DbMap
}

type PatientTxService struct {
	*Service
	DB *gorp.Transaction
}

// 病院内の患者一覧を取得する。
func (s *PatientService) List(
	hospitalId int,
	begin *time.Time,
	end *time.Time,
	limit int,
	offset int,
) ([]*model.Patient, int64, error) {
	if r, t, e := rds.ListPatients(s.DB, hospitalId, begin, end, limit, offset); e != nil {
		return nil, 0, C.DB_OPERATION_ERROR(e)
	} else {
		return r, t, nil
	}
}

// 指定した病院内の患者情報を取得する。
// TODO 取得内容が複雑化した場合、アクセスチェック用の関数と分ける。
func (s *PatientService) Fetch(
	id int,
	hospitalId int,
) (*model.Patient, error) {
	if r, e := rds.FetchPatient(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.HospitalId != hospitalId {
		return nil, C.NewNotFoundError(
			"patient_not_found",
			fmt.Sprintf("Patient %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// 医師からのアクセス可否を調べる。
func (s *PatientService) CheckAccessByDoctor(
	me *model.HospitalDoctor,
	id int,
) error {
	if r, e := rds.InquirePatient(s.DB, id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.HospitalId != me.Hospital.Id {
		return C.NewNotFoundError(
			"patient_not_found",
			fmt.Sprintf("Patient %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// 医師からのアクセス可否を調べる。
func (s *PatientService) CheckAccessByHospital(
	hospitalId int,
	id int,
) error {
	if r, e := rds.InquirePatient(s.DB, id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.HospitalId != hospitalId {
		return C.NewNotFoundError(
			"patient_not_found",
			fmt.Sprintf("Patient %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// 患者情報を更新する。
func (s *PatientTxService) Update(
	id int,
	name *string,
	age *int,
	numChildren *int,
	cesareanScar *bool,
	deliveryTime *int,
	bloodLoss *int,
	birthWeight *int,
	birthDatetime *time.Time,
	gestationalDays *int,
	apgarScore1Min *int,
	apgarScore5Min *int,
	umbilicalBlood *int,
	emergencyCesarean *bool,
	instrumentalLabor *bool,
	memo string,
) (*model.Patient, error) {
	var patient *model.Patient

	if r, e := rds.InquirePatient(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"patient_not_found",
			fmt.Sprintf("Patient %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		patient = r
	}

	patient.Name = name
	patient.Age = age
	patient.NumChildren = numChildren
	patient.CesareanScar = cesareanScar
	patient.DeliveryTime = deliveryTime
	patient.BloodLoss = bloodLoss
	patient.BirthWeight = birthWeight
	patient.BirthDatetime = birthDatetime
	patient.GestationalDays = gestationalDays
	patient.ApgarScore1Min = apgarScore1Min
	patient.ApgarScore5Min = apgarScore5Min
	patient.UmbilicalBlood = umbilicalBlood
	patient.EmergencyCesarean = emergencyCesarean
	patient.InstrumentalLabor = instrumentalLabor

	patient.Memo = memo
	patient.ModifiedAt = time.Now()

	if _, e := s.DB.Update(patient); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	patient, _ = rds.InquirePatient(s.DB, id)

	return patient, nil
}

// 医師からの更新可否を調べる。
func (s *PatientTxService) CheckUpdateByDoctor(
	me *model.HospitalDoctor,
	id int,
) error {
	if r, e := rds.InquirePatient(s.DB, id); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.HospitalId != me.Hospital.Id {
		return C.NewNotFoundError(
			"patient_not_found",
			fmt.Sprintf("Patient %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}
