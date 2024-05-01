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

type HospitalService struct {
	*Service
	DB *gorp.DbMap
}

type HospitalTxService struct {
	*Service
	DB *gorp.Transaction
}

// 病院をID順に取得する。
func (s *HospitalService) List(limit int, offset int) ([]*model.Hospital, int64, error) {
	if r, t, e := rds.ListHospitals(s.DB, limit, offset); e != nil {
		return nil, 0, C.DB_OPERATION_ERROR(e)
	} else {
		return r, t, nil
	}
}

// 病院をIDから取得する。
func (s *HospitalService) Fetch(id int) (*model.Hospital, error) {
	if r, e := rds.FetchHospital(s.DB, id); e != nil {
		return r, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"hospital_not_found",
			fmt.Sprintf("Hospital %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// 病院をIDから取得する。
func (s *HospitalService) InquireByUuid(uuid string) (*model.Hospital, error) {
	if r, e := rds.InquireHospitalByUuid(s.DB, uuid); e != nil {
		return r, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"hospital_not_found",
			fmt.Sprintf("Hospital %s is not found", uuid),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// 病院を登録する。
func (s *HospitalTxService) Create(name string, memo string) (*model.Hospital, error) {
	now := time.Now()

	hospitalUuid, _ := uuid.NewRandom()
	topic, _ := uuid.NewRandom()

	record := &model.Hospital{
		Uuid: hospitalUuid.String(),
		Topic: topic.String(),
		Name: name,
		Memo: memo,
		CreatedAt: now,
		ModifiedAt: now,
	}

	if e := s.DB.Insert(record); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	return record, nil
}

// 病院情報を更新する。
func (s *HospitalTxService) Update(id int, name string, memo string) (*model.Hospital, error) {
	record, err := inquireHospital(s.DB, id)

	if err != nil {
		return nil, err
	}

	now := time.Now()

	record.Name = name
	record.Memo = memo
	record.ModifiedAt = now

	if _, e := s.DB.Update(record); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	return record, nil
}

// 病院を削除する。
func (s *HospitalTxService) Delete(id int) error {
	record, err := inquireHospital(s.DB, id)

	if err != nil {
		return err
	}

	if _, e := s.DB.Delete(record); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	return nil
}

func inquireHospital(db model.QueryExecutor, id int) (*model.Hospital, error) {
	if r, e := rds.FetchHospital(db, id); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"hospital_not_found",
			fmt.Sprintf("Hospital %d is not found", id),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}