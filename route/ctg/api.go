package ctg

import (
	//"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	//C "github.com/spiker/spiker-server/constant"
	S "github.com/spiker/spiker-server/service"
	"github.com/spiker/spiker-server/route/shared"
)

func RegisterAPI(e *echo.Echo) {
	router := e.Group("/ctg")

	router.Use(
		middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
			service := shared.CreateService(S.CTGService{}, c).(*S.CTGService)

			ctg, err := service.Authenticate(key)

			if err != nil {
				return false, err
			} else if ctg == nil {
				return false, nil
			}

			c.Set(shared.ContextMeKey, ctg)

			return true, nil
		}),
	)

	registerAPIs(router)
}

func registerAPIs(router *echo.Group) {
	router.POST("/data", shared.C(uploadCTG))
}
