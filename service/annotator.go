package service

import (
	"time"

	"gopkg.in/gorp.v2"
	"github.com/google/uuid"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
)

type AnnotatorService struct {
	*Service
	DB *gorp.DbMap
}

type AnnotatorTxService struct {
	*Service
	DB *gorp.Transaction
}

// ログインIDとパスワードによる認証を行う。
func (s *AnnotatorService) Login(loginId string, password string) (*model.Annotator, error) {
	if r, e := rds.FetchAnnotatorByLoginId(s.DB, loginId, &password); e != nil {
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
func (s *AnnotatorService) Authenticate(authId string, version string) (*model.Annotator, error) {
	if r, e := rds.FetchAnnotatorByLoginId(s.DB, authId, nil); e != nil {
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
func (s *AnnotatorTxService) UpdateVersion(id int) error {
	now := time.Now()

	if version, e := uuid.NewRandom(); e != nil {
		return C.NewInternalServerError(
			"version_generaation_failed",
			"Failed to generate new version of your token",
			map[string]interface{}{},
		)
	} else if e := rds.UpdateAnnotatorTokenVersion(s.DB, id, version.String(), now); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else {
		return nil
	}
}