package route

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/spiker/spiker-server/route/admin"
	"github.com/spiker/spiker-server/route/annotation"
	"github.com/spiker/spiker-server/route/asset"
	"github.com/spiker/spiker-server/route/ctg"
	app_middleware "github.com/spiker/spiker-server/route/middleware"
	"github.com/spiker/spiker-server/route/monitor"
	"github.com/spiker/spiker-server/route/shared"
)

func NewHandler() *echo.Echo {
	e := echo.New()

	e.HTTPErrorHandler = shared.APIErrorHandler

	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())
	e.Use(middleware.RequestID())
	e.Use(app_middleware.SessionLogger)
	e.Use(app_middleware.I18n)
	e.Use(app_middleware.Transactional)

	admin.RegisterAPI(e)
	monitor.RegisterAPI(e)
	ctg.RegisterAPI(e)
	annotation.RegisterAPI(e)
	asset.RegisterAPI(e)

	return e
}
