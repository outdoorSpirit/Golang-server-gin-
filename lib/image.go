package lib

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"

	C "github.com/spiker/spiker-server/constant"

	"golang.org/x/image/draw"
)

type ImageConverter struct {
	Original image.Image
	Format   string
}

func NewImageConverter(src io.Reader) (*ImageConverter, error) {
	buf := new(bytes.Buffer)
	io.Copy(buf, src)
	_image, _format, err := image.Decode(buf)
	if err != nil {
		return nil, err
	}
	if _format != "jpeg" && _format != "png" && _format != "gif" {
		return nil, C.INVALID_IMAGE_FORMAT
	}
	return &ImageConverter{
		Original: _image,
		Format:   _format,
	}, nil
}

// Resize リサイズ
func (i *ImageConverter) Resize(size int) ([]byte, string, error) {
	rctSrc := i.Original.Bounds()
	var dstX, dstY int
	if rctSrc.Dx() > rctSrc.Dy() {
		dstX = size
		dstY = int(size * rctSrc.Dy() / rctSrc.Dx())
	} else {
		dstX = int(size * rctSrc.Dx() / rctSrc.Dy())
		dstY = size
	}
	imgDst := image.NewRGBA(image.Rect(0, 0, dstX, dstY))
	draw.CatmullRom.Scale(imgDst, imgDst.Bounds(), i.Original, rctSrc, draw.Over, nil)

	dst := new(bytes.Buffer)
	if err := png.Encode(dst, imgDst); err != nil {
		return nil, "", err
	}

	return dst.Bytes(), "png", nil
}
