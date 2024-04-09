package admin

import (
	//"fmt"
	//"database/sql"
	"net/http"
	"net/http/httptest"
	//"net/url"
	"testing"
	//"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"

	//"github.com/spiker/spiker-server/route/shared"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/test"
	F "github.com/spiker/spiker-server/test/fixture"
)

func TestAdminLogin_Login(t *testing.T) {
	(&F.AdminFixture{}).Generate(3)

	httpTests := test.HttpTests{
		{
			Name:    "ログイン成功",
			Method:  http.MethodPost,
			Path:    "/admin/login",
			Body:    test.JsonBody(map[string]interface{}{
				"loginId": "admin-0002",
				"password": "pass-0002",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				res := F.FromJsonResponse(t, rec, &loginResponse{}).(*loginResponse)

				token, err := jwt.Parse(res.AccessToken, func(t *jwt.Token) (interface{}, error) {
					return []byte(lib.GetSecret()), nil
				})
				assert.NoError(t, err)

				claims := token.Claims.(jwt.MapClaims)

				assert.EqualValues(t, "admin-0002", claims["sub"])
				assert.EqualValues(t, "token-0002", claims["jti"])
				assert.EqualValues(t, "SPIKER-SERVER-TEST", claims["iss"])
				assert.EqualValues(t, "SPIKER", claims["aud"])
			},
		},
		{
			Name:    "ログイン失敗",
			Method:  http.MethodPost,
			Path:    "/admin/login",
			Body:    test.JsonBody(map[string]interface{}{
				"loginId": "admin-0002",
				"password": "pass-0000",
			}),
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, nil)
}

func TestAdminLogin_Logout(t *testing.T) {
	auth := (&F.AdminFixture{}).Generate(3)

	db := lib.GetDB(lib.WriteDBKey)

	httpTests := test.HttpTests{
		{
			Name:    "トークン無効化",
			Method:  http.MethodDelete,
			Path:    "/admin/login",
			Token:   auth.Token(1),
			Prepare: func(req *http.Request) {
				if _, e := db.Exec(`UPDATE administrator SET token_version = 'token-0002' where id = 2`); e != nil {
					assert.FailNow(t, e.Error())
				}
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, rec.Code)

				var actual *model.Administrator
				if r, e := db.Get(model.Administrator{}, 2); e != nil {
					assert.FailNow(t, e.Error())
				} else {
					assert.NotNil(t, r)
					actual = r.(*model.Administrator)
				}

				assert.NotEqual(t, "token-0002", actual.TokenVersion)
			},
		},
		{
			Name:    "無効化を検証",
			Method:  http.MethodDelete,
			Path:    "/admin/login",
			Token:   auth.Token(1),
			Prepare: func(req *http.Request) {
				if _, e := db.Exec(`UPDATE administrator SET token_version = 'unmatch' where id = 2`); e != nil {
					assert.FailNow(t, e.Error())
				}
			},
			Check:   func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	httpTests.Run(testHandler(), t, nil)
}