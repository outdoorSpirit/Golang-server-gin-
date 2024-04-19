package service

import (
	"time"

	"gopkg.in/gorp.v2"
	"github.com/google/uuid"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
)

type AdministratorService struct {
	*Service
	DB *gorp.DbMap
}

type AdministratorTxService struct {
	*Service
	DB *gorp.Transaction
}

// ログインIDとパスワードによる認証を行う。
func (s *AdministratorService) Login(loginId string, password string) (*model.Administrator, error) {
	if r, e := rds.FetchAdministratorByLoginId(s.DB, loginId, &password); e != nil {
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
func (s *AdministratorService) Authenticate(authId string, version string) (*model.Administrator, error) {
	if r, e := rds.FetchAdministratorByLoginId(s.DB, authId, nil); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.TokenVersion != version {
		return nil, C.NewUnauthorizedError(
			"unauthorized",
			"Your token is not valid",
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

// トークンバージョンを更新する。
func (s *AdministratorTxService) UpdateVersion(id int) error {
	now := time.Now()

	if version, e := uuid.NewRandom(); e != nil {
		return C.NewInternalServerError(
			"version_generaation_failed",
			"Failed to generate new version of your token",
			map[string]interface{}{},
		)
	} else if e := rds.UpdateAdministratorTokenVersion(s.DB, id, version.String(), now); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else {
		return nil
	}
}

func (s *AdministratorTxService) Create(loginId string, password string) (*model.Administrator, error) {
	now := time.Now()

	admin := &model.Administrator{
		LoginId: loginId,
		Password: password,
		TokenVersion: "",
		CreatedAt: now,
		ModifiedAt: now,
	}

	if e := s.DB.Insert(admin); e != nil {
		return nil, e
	}

	if e := rds.HashAdministratorPassword(s.DB, admin.Id); e != nil {
		return nil, e
	}

	return admin, nil
}