package shared

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/spiker/spiker-server/lib"
	"gopkg.in/gorp.v2"
)

const (
	contextTxDatabaseKey    string = "tx_db"
	contextReadDatabaseKey         = "read_db"
	ContextSessionLoggerKey        = "session_logger"
	ContextFirebaseTokenKey        = "firebase_token"
	ContextI18NLangKey             = "lang_key"
	ContextMeKey                   = "me"
)

const (
	HeaderAcceptLanguage = "Accept-Language"
	HeaderXAccelRedirect = "X-Accel-Redirect"
)

var (
	cacheObj        *cache.Cache
	CurrentLocation *time.Location
)

func init() {
	cacheObj = cache.New(1*time.Minute, 15*time.Minute)
	CurrentLocation, _ = time.LoadLocation("Asia/Tokyo")
}

type Context struct {
	echo.Context
}

type contextFunc func(c *Context) error

// C カスタマイズしたコンテキストをWrapする
func C(ctxFunc contextFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return ctxFunc(c.(*Context))
	}
}

func (c *Context) GetCache() *cache.Cache {
	return cacheObj
}

func (c *Context) GetTransaction() *gorp.Transaction {
	db := c.Get(contextTxDatabaseKey)
	if db != nil {
		return db.(*gorp.Transaction)
	}
	_db := lib.GetDB(lib.WriteDBKey)
	if tx, err := _db.Begin(); err != nil {
		return nil
	} else {
		c.Set(contextTxDatabaseKey, tx)
		return tx
	}
}

func (c *Context) GetReadDB() *gorp.DbMap {
	db := c.Get(contextReadDatabaseKey)
	if db != nil {
		return db.(*gorp.DbMap)
	}
	_db := lib.GetDB(lib.ReadDBKey)
	c.Set(contextReadDatabaseKey, _db)
	return _db
}

func (c *Context) Commit() {
	db := c.Get(contextTxDatabaseKey)
	if db != nil {
		if _db, ok := db.(*gorp.Transaction); ok {
			_db.Commit()
		}
	}
}

func (c *Context) Rollback() {
	db := c.Get(contextTxDatabaseKey)
	if db != nil {
		if _db, ok := db.(*gorp.Transaction); ok {
			_db.Rollback()
		}
	}
}

func (c *Context) IntParam(key string) int {
	param := c.Param(key)
	if len(param) > 0 {
		intParam, err := strconv.Atoi(param)
		if err != nil {
			return 0
		} else {
			return intParam
		}
	} else {
		return 0
	}
}

func (c *Context) LocalTime() time.Time {
	return time.Now().In(CurrentLocation)
}

func (c *Context) DateToTimeRangeUTC(date time.Time) (time.Time, time.Time) {
	begin := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0.0, CurrentLocation)
	end := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0.0, CurrentLocation)
	return begin.UTC(), end.UTC()
}

func (c *Context) Log() *logrus.Entry {
	return c.Get(ContextSessionLoggerKey).(*logrus.Entry)
}
