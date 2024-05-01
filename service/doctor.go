package service

import (
	"fmt"
	"time"

	"gopkg.in/gorp.v2"
	"github.com/google/uuid"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
)

type DoctorService struct {
	*Service
	DB *gorp.DbMap
}

type DoctorTxService struct {
	*Service
	DB *gorp.Transaction
}

// ログインIDとパスワードによる認証を行う。
func (s *DoctorService) Login(loginId string, password string) (*model.HospitalDoctor, error) {
	if r, e := rds.FetchDoctorByLoginId(s.DB, loginId, &password); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewUnauthorizedError(
			"unauthorized",
			"Login ID or password is not correct",
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// トークン認証を行う。
func (s *DoctorService) Authenticate(authId string, version string) (*model.HospitalDoctor, error) {
	if r, e := rds.FetchDoctorByLoginId(s.DB, authId, nil); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.Doctor.TokenVersion != version {
		return nil, C.NewUnauthorizedError(
			"unauthorized",
			"Your token is not valid",
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// 病院内の医者をID順に取得する。
func (s *DoctorService) List(hospitalId int, limit int, offset int) ([]*model.Doctor, int64, error) {
	if r, t, e := rds.ListDoctorsInHospital(s.DB, hospitalId, limit, offset); e != nil {
		return nil, 0, C.DB_OPERATION_ERROR(e)
	} else {
		return r, t, nil
	}
}

// 病院と医者の関連を調べる。
func (s *DoctorService) CheckDoctorInHospital(id int, hospitalId int) error {
	if ok, e := rds.CheckDoctorInHospital(s.DB, id, hospitalId); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if !ok {
		return C.NewNotFoundError(
			"doctor_not_found",
			fmt.Sprintf("Doctor %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

// 医者情報を取得する。
func (s *DoctorService) Fetch(id int) (*model.Doctor, error) {
	if r, e := rds.FetchDoctor(s.DB, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"doctor_not_found",
			fmt.Sprintf("Doctor %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// トークンバージョンを更新する。
func (s *DoctorTxService) UpdateVersion(id int) error {
	now := time.Now()

	if version, e := uuid.NewRandom(); e != nil {
		return C.NewInternalServerError(
			"version_generaation_failed",
			"Failed to generate new version of your token",
			map[string]interface{}{},
		)
	} else if e := rds.UpdateDoctorTokenVersion(s.DB, id, version.String(), now); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else {
		return nil
	}
}

// 医者を病院に登録する。
func (s *DoctorTxService) Create(hospitalId int, loginId string, password string, name string) (*model.Doctor, error) {
	now := time.Now()

	if h, e := rds.FetchHospital(s.DB, hospitalId); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if h == nil {
		return nil, C.NewNotFoundError(
			"hospital_not_found",
			fmt.Sprintf("Hospital %d is not found", hospitalId),
			map[string]interface{}{},
		)
	}

	if d, e := rds.FetchDoctorByLoginId(s.DB, loginId, nil); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if d != nil {
		return nil, C.NewBadRequestError(
			"login_id_exists",
			fmt.Sprintf("Login ID %s is alredy registered", loginId),
			map[string]interface{}{},
		)
	}

	version, _ := uuid.NewRandom()
	topic, _ := uuid.NewRandom()

	record := &model.Doctor{
		HospitalId: hospitalId,
		LoginId: loginId,
		Password: "",
		TokenVersion: version.String(),
		Topic: topic.String(),
		Name: name,
		CreatedAt: now,
		ModifiedAt: now,
	}

	if e := s.DB.Insert(record); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	if e := rds.SetDoctorPassword(s.DB, record.Id, password); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	return record, nil
}

// 医者情報を更新する。
func (s *DoctorTxService) Update(id int, name string) (*model.Doctor, error) {
	record, err := inquireDoctor(s.DB, id)

	if err != nil {
		return nil, err
	}

	now := time.Now()

	record.Name = name
	record.ModifiedAt = now

	if _, e := s.DB.Update(record); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	return record, nil
}

// 医者のパスワードを更新して既存のトークンを無効化する。
func (s *DoctorTxService) UpdatePassword(id int, password string) error {
	record, err := inquireDoctor(s.DB, id)

	if err != nil {
		return err
	}

	now := time.Now()

	version, _ := uuid.NewRandom()

	record.TokenVersion = version.String()
	record.Password = password
	record.ModifiedAt = now

	if _, e := s.DB.Update(record); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	if e := rds.SetDoctorPassword(s.DB, id, password); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	return nil
}

// 医者を削除する。
func (s *DoctorTxService) Delete(id int) error {
	record, err := inquireDoctor(s.DB, id)

	if err != nil {
		return err
	}

	if _, e := s.DB.Delete(record); e != nil {
		return e
	}

	return nil
}

func inquireDoctor(db model.QueryExecutor, id int) (*model.Doctor, error) {
	if r, e := rds.FetchDoctor(db, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"doctor_not_found",
			fmt.Sprintf("Doctor %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}