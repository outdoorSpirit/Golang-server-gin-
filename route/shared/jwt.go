package shared

import (
	"github.com/golang-jwt/jwt"

	"github.com/spiker/spiker-server/lib"
)

func CreateTokenWithMapClaims(claims jwt.MapClaims) string {
	cfg := lib.GetJWTConfiguration()

	claims["aud"] = cfg.Audience
	claims["iss"] = cfg.Issuer

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, _ := token.SignedString([]byte(cfg.Secret))

	return t
}

func CreateTokenWithStandardClaims(subject string, tokenId string) string {
	cfg := lib.GetJWTConfiguration()

	claims := &jwt.StandardClaims{
		Issuer:   cfg.Issuer,
		Audience: cfg.Audience,
		Subject:  subject,
	}

	if tokenId != "" {
		claims.Id = tokenId
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, _ := token.SignedString([]byte(cfg.Secret))

	return t
}
