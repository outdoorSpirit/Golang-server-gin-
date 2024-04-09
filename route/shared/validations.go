package shared

import (
	"bytes"
	"image"
)

func ValidateImage(body []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(body))
	return format, err
}
