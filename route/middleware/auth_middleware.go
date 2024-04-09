package middleware

import (
	"regexp"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	authScheme string = "Bearer"
)

func jwtFromHeader(c echo.Context, header string, authScheme string) (string, error) {
	auth := c.Request().Header.Get(header)
	l := len(authScheme)
	if len(auth) > l+1 && auth[:l] == authScheme {
		return auth[l+1:], nil
	}
	return "", middleware.ErrJWTMissing
}

// SkipPattern Middlewareを実行しないパターン。
type SkipPattern struct {
	Methods   []string
	PathRegex *regexp.Regexp
}

type SkipPatterns []SkipPattern

func (p SkipPatterns) Match(c echo.Context) bool {
	for _, pattern := range p {
		for _, method := range pattern.Methods {
			if method == c.Request().Method {
				if matched := pattern.PathRegex.MatchString(c.Request().URL.Path); matched {
					return true
				}
			}
		}
	}
	return false
}
