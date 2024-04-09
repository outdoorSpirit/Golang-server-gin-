package annotation

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
	e.POST("/annotation/login", shared.C(login))

	router := e.Group("/annotation")

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
					service := shared.CreateService(S.AnnotatorService{}, c).(*S.AnnotatorService)

					annotator, err := service.Authenticate(authId, version)

					return annotator, err
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

	// 病院。
	router.GET("/hospitals", shared.C(listHospitals))
	router.GET("/hospitals/:hospital_uuid", shared.C(fetchHospital))

	// データ。
	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id/heartrates", shared.C(listHeartRate))
	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id/tocos", shared.C(listTOCO))

	// 患者。
	router.GET("/hospitals/:hospital_uuid/patients", shared.C(listPatients))
	router.GET("/hospitals/:hospital_uuid/patients/:patient_id", shared.C(fetchPatient))

	// 計測記録。
	router.GET("/hospitals/:hospital_uuid/measurements", shared.C(listMeasurements))
	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id", shared.C(fetchMeasurement))
	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id/silent", shared.C(getSilentState))

	// イベント。
	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id/events", shared.C(listComputedEvents))
	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id/events/:event_id", shared.C(fetchComputedEvent))
	router.POST("/hospitals/:hospital_uuid/measurements/:measurement_id/events/:event_id/show", shared.C(showComputedEvent))
	router.DELETE("/hospitals/:hospital_uuid/measurements/:measurement_id/events/:event_id/show", shared.C(hideComputedEvent))
	router.POST("/hospitals/:hospital_uuid/measurements/:measurement_id/events/:event_id/suspend", shared.C(suspendComputedEvent))

	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id/annotations", shared.C(listAnnotatedEvents))
	router.GET("/hospitals/:hospital_uuid/measurements/:measurement_id/annotations/:annotation_id", shared.C(fetchAnnotatedEvent))
	router.POST("/hospitals/:hospital_uuid/measurements/:measurement_id/annotations", shared.C(registerAnnotatedEvent))
	router.PUT("/hospitals/:hospital_uuid/measurements/:measurement_id/annotations/:annotation_id", shared.C(updateAnnotatedEvent))
	router.DELETE("/hospitals/:hospital_uuid/measurements/:measurement_id/annotations/:annotation_id", shared.C(deleteAnnotatedEvent))

	// アラート。
	router.GET("/hospitals/:hospital_uuid/alerts", shared.C(collectUnreadAlerts))
}
