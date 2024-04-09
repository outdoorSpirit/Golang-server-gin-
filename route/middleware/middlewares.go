package middleware

import (
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"

	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/route/shared"
)

func Transactional(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := &shared.Context{c}
		var err error
		err = next(cc)
		if e := recover(); e != nil {
			cc.Rollback()
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("panic occured")
			err = e.(error)
		} else if err != nil {
			cc.Rollback()
		} else {
			cc.Commit()
		}
		return err
	}
}

func SessionLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := log.WithFields(log.Fields{"request_id": c.Response().Header().Get(echo.HeaderXRequestID)})
		c.Set(shared.ContextSessionLoggerKey, logger)
		return next(c)
	}
}

func I18n(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		acceptLang := c.Request().Header.Get(shared.HeaderAcceptLanguage)
		paramLang := c.QueryParam("lang")

		localizer := lib.NewLocalizer(paramLang, acceptLang)
		c.Set(shared.ContextI18NLangKey, localizer)

		return next(c)
	}
}
