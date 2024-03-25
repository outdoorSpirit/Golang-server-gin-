package lib

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImage_Resize(t *testing.T) {
	root := os.Getenv("SERVER_ROOT")
	src, err := os.Open(path.Join(root, "test/data/test_image.jpg"))
	assert.NoError(t, err)
	assert.NotNil(t, src)
	defer src.Close()

	conv, err := NewImageConverter(src)
	assert.NoError(t, err)
	bytes, format, err := conv.Resize(500)
	assert.NoError(t, err)
	assert.Equal(t, "png", format)
	assert.NotEqual(t, 0, len(bytes))

	err = ioutil.WriteFile(path.Join(root, "test/data/output_test_image.png"), bytes, 0644)
	assert.NoError(t, err)
}
