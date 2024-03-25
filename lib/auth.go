package lib

import (
	"fmt"

	"github.com/golang-jwt/jwt"
	jwt_ "github.com/dgrijalva/jwt-go"

	C "github.com/spiker/spiker-server/constant"
)

type JWTConfiguration struct {
	Secret   string
	Issuer   string
	Audience string
}

func (cfg *JWTConfiguration) String() string {
	return fmt.Sprintf(`[JWT]
Issuer:   %v
Audience: %v`, cfg.Issuer, cfg.Audience)
}

var (
	defaultConfiguration *JWTConfiguration = nil
)

func SetupAuthentication(config *JWTConfiguration) error {
	defaultConfiguration = config

	return nil
}

func GetJWTConfiguration() *JWTConfiguration {
	return defaultConfiguration
}

func GetSecret() string {
	return defaultConfiguration.Secret
}

func VerifyToken(token *jwt.Token) (string, error) {
	if !token.Valid {
		return "", C.NewUnauthorizedError(
			"invalid_token",
			"Parsed token is invalid",
			map[string]interface{}{},
		)
	}

	claims := token.Claims.(jwt.MapClaims)

	if !claims.VerifyIssuer(defaultConfiguration.Issuer, true) {
		return "", C.NewUnauthorizedError(
			"invalid_issuer",
			fmt.Sprintf("The issuer in your token is invalid: %v", claims["iss"]),
			map[string]interface{}{},
		)
	}

	if !claims.VerifyAudience(defaultConfiguration.Audience, true) {
		return "", C.NewUnauthorizedError(
			"invalid_audience",
			fmt.Sprintf("The audience in your token is invalid: %v", claims["aud"]),
			map[string]interface{}{},
		)
	}

	if sub, be := claims["sub"]; !be {
		return "", C.NewUnauthorizedError(
			"subject_not_found",
			"The subject is not found in your token",
			map[string]interface{}{},
		)
	} else if authId, ok := sub.(string); !ok {
		return "", C.NewUnauthorizedError(
			"invalid_subject",
			fmt.Sprintf("The subject in your token is invalid: %v", sub),
			map[string]interface{}{},
		)
	} else {
		return authId, nil
	}
}

// 古いライブラリバージョンのJWTを新しいバージョンに変換した上で、認証IDとトークンバージョンを利用した認証を適用する。
func ConvertAndAuthenticate(oldToken *jwt_.Token, authenticate func(string, string)(interface{}, error)) (interface{}, error) {
	token := &jwt.Token{
		Raw:       oldToken.Raw,
		Method:    oldToken.Method,
		Header:    oldToken.Header,
		Claims:    (jwt.MapClaims)((map[string]interface{})(oldToken.Claims.(jwt_.MapClaims))),
		Signature: oldToken.Signature,
		Valid:     oldToken.Valid,
	}

	var version string

	if jti, be := token.Claims.(jwt.MapClaims)["jti"]; !be {
		version = ""
	} else if ver, ok := jti.(string); !ok {
		return nil, fmt.Errorf("jti claim is not a string")
	} else {
		version = ver
	}

	authId, err := VerifyToken(token)

	if err != nil {
		return nil, err
	}

	return authenticate(authId, version)
}