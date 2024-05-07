package service

import (
	"os"
	""
)

func init() {
	os.Setenv("SERVER_ENV", "test")
	config.SetupAll()
}
