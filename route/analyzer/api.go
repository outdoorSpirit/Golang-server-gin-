package analyzer

/*
import (
	"github.com/labstack/echo/v4"

	app_middleware "github.com/spiker/spiker-server/route/middleware"
	"github.com/spiker/spiker-server/route/shared"
)

func checkAnalyzerUser(skips app_middleware.SkipPatterns) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
}

func RegisterAPI(e *echo.Echo) {
	router := e.Group("/analyzer")

	router.Use(checkAnalyzerUser(
		app_middleware.SkipPatterns([]app_middleware.SkipPattern{})),
	)

	registerAPIs(router)
}

func registerAPIs(router *echo.Group) {
	// 診断。
	router.POST("/hospitals/:hospital_uuid/patients/:patient_code/measurements", shared.C(registerMeasurement))
	router.PUT("/hospitals/:hospital_uuid/patients/:patient_code/measurements/:measurement_id", shared.C(updateMeasurement))
	router.DELETE("/hospitals/:hospital_uuid/patients/:patient_code/measurements/:measurement_id", shared.C(deleteMeasurement))
}
*/
