package admin

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
	e.POST("/admin/login", shared.C(login))

	router := e.Group("/admin")

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
					service := shared.CreateService(S.AdministratorService{}, c).(*S.AdministratorService)

					admin, err := service.Authenticate(authId, version)

					return admin, err
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
	router.POST("/hospitals", shared.C(createHospital))
	router.GET("/hospitals/:hospital_id", shared.C(fetchHospital))
	router.PUT("/hospitals/:hospital_id", shared.C(updateHospital))
	router.DELETE("/hospitals/:hospital_id", shared.C(deleteHospital))
	router.POST("/hospitals/:hospital_id/api_key", shared.C(generateApiKey))

	// 医者。
	router.GET("/hospitals/:hospital_id/doctors", shared.C(listDoctors))
	router.POST("/hospitals/:hospital_id/doctors", shared.C(createDoctor))
	router.GET("/hospitals/:hospital_id/doctors/:doctor_id", shared.C(fetchDoctor))
	router.PUT("/hospitals/:hospital_id/doctors/:doctor_id", shared.C(updateDoctor))
	router.PUT("/hospitals/:hospital_id/doctors/:doctor_id/password", shared.C(updateDoctorPassword))
	router.DELETE("/hospitals/:hospital_id/doctors/:doctor_id", shared.C(deleteDoctor))
}
