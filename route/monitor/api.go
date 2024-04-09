package monitor

import (
	//"fmt"

	"github.com/labstack/echo/v4"
	//"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4/middleware"
	jwt_ "github.com/dgrijalva/jwt-go"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
	S "github.com/spiker/spiker-server/service"
	"github.com/spiker/spiker-server/route/shared"
)

const (
	jwtTokenKey = "token"
)

func RegisterAPI(e *echo.Echo) {
	e.POST("/1/login", shared.C(login))

	router := e.Group("/1")

	router.Use(
		middleware.JWTWithConfig(middleware.JWTConfig{
			ContextKey: jwtTokenKey,
			SigningKey: []byte(lib.GetSecret()),
		}),
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				oldToken, ok := c.Get(jwtTokenKey).(*jwt_.Token)

				if !ok {
					return C.NewUnauthorizedError(
						"token_not_found",
						"Token was parsed but missed unexpectedly",
						map[string]interface{}{},
					)
				}

				me, err := lib.ConvertAndAuthenticate(oldToken, func(authId string, version string) (interface{}, error) {
					service := shared.CreateService(S.DoctorService{}, c).(*S.DoctorService)

					doctor, err := service.Authenticate(authId, version)

					return doctor, err
				})

				if err != nil {
					if e, ok := err.(C.AppError); ok {
						return e
					} else {
						return C.NewUnauthorizedError(
							"Unauthorized",
							e.Error(),
							map[string]interface{}{},
						)
					}
				}

				c.Set(shared.ContextMeKey, me)

				return next(c)
			}
		},
	)

	registerAPIs(router)
}

func registerAPIs(router *echo.Group) {
	// ログイン。
	router.DELETE("/login", shared.C(logout))

	// 患者。
	router.GET("/patients", shared.C(listPatients))
	router.GET("/patients/:patient_id", shared.C(fetchPatient))
	router.PUT("/patients/:patient_id", shared.C(updatePatient))

	// 計測端末。
	router.GET("/terminals", shared.C(listTerminals))
	router.GET("/terminals/:terminal_id", shared.C(fetchTerminal))
	router.PUT("/terminals/:terminal_id", shared.C(updateTerminal))

	// 計測記録。
	router.GET("/measurements", shared.C(listMeasurements))
	router.GET("/measurements/:measurement_id", shared.C(fetchMeasurement))
	router.GET("/measurements/:measurement_id/silent", shared.C(getSilentState))
	router.POST("/measurements/:measurement_id/silent", shared.C(setSilentState))
	router.POST("/measurements/:measurement_id/close", shared.C(closeMeasurement))

	// イベント。
	router.GET("/measurements/:measurement_id/events", shared.C(listComputedEvents))
	router.GET("/measurements/:measurement_id/events/:event_id", shared.C(fetchComputedEvent))
	router.GET("/measurements/:measurement_id/annotations", shared.C(listAnnotatedEvents))
	router.GET("/measurements/:measurement_id/annotations/:annotation_id", shared.C(fetchAnnotatedEvent))
	router.POST("/measurements/:measurement_id/annotations/:annotation_id/close", shared.C(closeAnnotatedEvent))

	// データ。
	router.GET("/measurements/:measurement_id/heartrates", shared.C(listHeartRate))
	router.GET("/measurements/:measurement_id/tocos", shared.C(listTOCO))

	// アラート。
	router.GET("/alerts", shared.C(collectAlerts))
}
