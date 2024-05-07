package service

import (
	"os"
	"github.com/spiker/spiker-server/config"
)

func init() {
	os.Setenv("SERVER_ENV", "test")
	config.SetupAll()
}
