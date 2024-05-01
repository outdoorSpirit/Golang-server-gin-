package monitor

import (
	"github.com/labstack/echo/v4"

	"github.com/spiker/spiker-server/test"
)

func testHandler() *echo.Echo {
	e := test.TestHandler()

	RegisterAPI(e)

	return e
}