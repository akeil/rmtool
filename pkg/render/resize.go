package render

import (
	"image"

	"golang.org/x/image/draw"
)

// separate file because  we want to import x/image/draw
// instead of image/draw.

func resize(i image.Image, r image.Rectangle) image.Image {
	dst := image.NewRGBA(r)
	s := draw.BiLinear
	s.Scale(dst, r, i, i.Bounds(), draw.Over, nil)
	return dst
}
