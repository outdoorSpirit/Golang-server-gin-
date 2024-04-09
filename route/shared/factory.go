package shared

import (
	"os"
	"path"
	"reflect"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/spiker/spiker-server/lib"
	S "github.com/spiker/spiker-server/service"
	"gopkg.in/gorp.v2"
)

const (
	dbStructTag string = "db"
)

var servicePackagePath string = ""

func init() {
	servicePackagePath = path.Join(os.Getenv("SERVER_ROOT"), "service")
}

func CreateService(obj interface{}, c echo.Context) interface{} {
	cc, ok := c.(*Context)
	if !ok {
		return nil
	}

	t := reflect.TypeOf(obj)

	if t.PkgPath() == servicePackagePath {
		return nil
	}

	v := reflect.New(t)
	e := v.Elem()

	for i := 0; i < e.NumField(); i++ {
		valueField := e.Field(i)
		typeField := t.Field(i)
		valueType := typeField.Type

		if valueType.Kind() == reflect.Ptr {
			valueType = valueType.Elem()
		}
		if valueType == reflect.TypeOf(gorp.DbMap{}) {
			valueField.Set(reflect.ValueOf(
				cc.GetReadDB(),
			))
		} else if valueType == reflect.TypeOf(gorp.Transaction{}) {
			valueField.Set(reflect.ValueOf(
				cc.GetTransaction(),
			))
		} else if valueType == reflect.TypeOf((*lib.InfluxDBClient)(nil)).Elem() {
			valueField.Set(reflect.ValueOf(
				lib.GetInfluxDB(),
			))
		} else if valueType == reflect.TypeOf(lib.Localizer{}) {
			localizer := c.Get(ContextI18NLangKey).(*lib.Localizer)
			valueField.Set(reflect.ValueOf(localizer))
		} else if valueType == reflect.TypeOf(S.Service{}) {
			_logger := c.Get(ContextSessionLoggerKey)
			if _logger != nil {
				logger := _logger.(*logrus.Entry)
				_service := &S.Service{
					Log: logger,
				}
				valueField.Set(reflect.ValueOf(
					_service,
				))
			}
		}
	}

	return v.Interface()
}
