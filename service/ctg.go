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

type CTGService struct {
	*Service
	DB *gorp.DbMap
}

type CTGTxService struct {
	*Service
	DB *gorp.Transaction
}

// APIキーから認証を行う。
func (s *CTGService) Authenticate(apiKey string) (*model.CTGAuthentication, error) {
	if r, e := rds.FetchCTGByApiKey(s.DB, apiKey); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewUnauthorizedError(
			"invalid_api_key",
			"Your API key does not exists",
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// 病院に対するAPIキーを発行する。
func (s *CTGTxService) GenerateApiKey(hospitalId int) (*model.CTGAuthentication, error) {
	if h, e := s.DB.Get(model.Hospital{}, hospitalId); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if h == nil {
		return nil, C.NewNotFoundError(
			"hospital_not_found",
			fmt.Sprintf("Hospital #%d is not found", hospitalId),
			map[string]interface{}{},
		)
	}

	apiKey, _ := uuid.NewRandom()

	record := &model.CTGAuthentication{
		HospitalId: hospitalId,
		ApiKey: apiKey.String(),
		CreatedAt: time.Now(),
	}

	if e := s.DB.Insert(record); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	return record, nil
}