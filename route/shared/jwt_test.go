package shared

import (
	"os"
	"testing"

	"github.com/spiker/spiker-server/config"
	"github.com/stretchr/testify/assert"
)

func Test_CreateTokenWithMapClaims(t *testing.T) {
	os.Setenv("SERVER_ENV", "test")
	config.SetupAll()

	claims := map[string]interface{}{
		"uuid": "test-uuid",
		"scopes": []string{
			"bim:read",
		},
	}

	token := CreateTokenWithMapClaims(claims)
	assert.Less(t, 0, len(token))
}

func Test_CreateTokenWithStandardClaims(t *testing.T) {
	os.Setenv("SERVER_ENV", "test")
	config.SetupAll()

	token := CreateTokenWithStandardClaims(`uuid2`, "")
	assert.Less(t, 0, len(token))
}
