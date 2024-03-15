package config

import (
	"testing"

	"github.com/spiker/spiker-server/lib"
	"github.com/stretchr/testify/assert"
)

func TestEnv_Load(t *testing.T) {
	SetupAll()
	server := ServerConfig()

	assert.Equal(t, ":1323", server.Port)
	assert.Equal(t, true, server.Dump)

	db := appConfig.DB
	assert.Equal(t, "db", db.Host)
	assert.Equal(t, 5432, db.Port)
	assert.Equal(t, "postgres", db.User)
	assert.Equal(t, "postgres", db.Password)
	assert.Equal(t, "spiker", db.Name)
	assert.Equal(t, 10, db.Maxconns)
	assert.Equal(t, 10, db.Maxidles)

	assert.Equal(t, "data/l10n", appConfig.Lang.Path)

	db = appConfig.ReadDB
	assert.Equal(t, "db", db.Host)
	assert.Equal(t, 5432, db.Port)
	assert.Equal(t, "postgres", db.User)
	assert.Equal(t, "postgres", db.Password)
	assert.Equal(t, "spiker", db.Name)
	assert.Equal(t, 10, db.Maxconns)
	assert.Equal(t, 10, db.Maxidles)
}

func TestEnv_Localizer(t *testing.T) {
	localizer := lib.NewLocalizer("ja")
	assert.Equal(t, "こんにちは、世界!", localizer.Localize("hello world!", nil))

	localizer = lib.NewLocalizer("en")
	assert.Equal(t, "Hello World!", localizer.Localize("hello world!", nil))
}
