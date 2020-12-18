package render

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"

	"akeil.net/akeil/rm"
)

var colors = map[rm.BrushColor]color.Color{
	rm.Black: color.Black,
	rm.Gray:  color.RGBA{127, 127, 127, 255},
	rm.White: color.White,
}
var bgColor = color.White

// RenderDrawing paints the given drawing and writes the result to the given
// writer.
func RenderDrawing(d *rm.Drawing, w io.Writer) error {
	err := RenderPNG(d, w)
	if err != nil {
		return err
	}

	return nil
}

// RenderPNG dapints the given drawing to a PNG file and writes the PNG data
// to the given writer.
func RenderPNG(d *rm.Drawing, w io.Writer) error {
	// TODO: use width/height from Drawing/Metadata
	r := image.Rect(0, 0, 1404, 1872)
	dst := image.NewRGBA(r)

	renderBackground(dst)

	for _, l := range d.Layers {
		err := renderLayer(dst, l)
		if err != nil {
			return err
		}
	}

	err := png.Encode(w, dst)
	if err != nil {
		return err
	}

	return nil
}

// renderBackground fills the complete destination image with the background color (white).
func renderBackground(dst draw.Image) {
	b := dst.Bounds()
	x0 := b.Min.X
	x1 := b.Max.X
	y0 := b.Min.Y
	y1 := b.Max.Y

	for x := x0; x < x1; x++ {
		for y := y0; y < y1; y++ {
			dst.Set(x, y, bgColor)
		}
	}
}

// renderLayer paints all strokes from the given layer onto the destination image.
func renderLayer(dst draw.Image, l rm.Layer) error {
	for _, s := range l.Strokes {
		// The erased content is deleted,
		// but eraser strokes are recorded.
		if s.BrushType == rm.Eraser {
			continue
		}

		err := renderStroke(dst, s)
		if err != nil {
			return err
		}
	}

	return nil
}

// renderStroke paints a single stroke on the destination image..
func renderStroke(dst draw.Image, s rm.Stroke) error {
	pen := NewBrush(s.BrushType)
	mask, err := loadBrushMask(pen)
	if err != nil {
		return err
	}

	c := colors[s.BrushColor]
	if c == nil {
		return fmt.Errorf("invalid color %v", s.BrushColor)
	}
	color := image.NewUniform(c)

	numDots := len(s.Dots)
	for i := 1; i < numDots; i++ {
		start := s.Dots[i-1]
		end := s.Dots[i]
		renderSegment(dst, mask, color, pen, start, end)
	}

	return nil
}

// renderSegment places stamps along the path from start to end dots.
// Stamps are spaced evenly and overlap.
func renderSegment(dst draw.Image, mask image.Image, color image.Image, pen Brush, start, end rm.Dot) {
	// Scale the image according to the brush width
	width := float64(start.Width)
	scaledSize := int(math.Round(width))
	scale := image.Rect(0, 0, scaledSize, scaledSize)
	scaled := resize(mask, scale)

	// Apply additional opacity for pressure/speed
	opacity := pen.Opacity(start.Pressure, start.Speed)
	opaque := applyOpacity(scaled, opacity)

	// Rotate the brush to align with the path
	angle := math.Atan2(float64(start.Y-end.Y), float64(start.X-end.X))
	rotated := rotate(angle, opaque)

	w, h := scaledSize, scaledSize
	overlap := pen.Overlap()

	a := math.Abs(float64(start.Y - end.Y))
	b := math.Abs(float64(start.X - end.X))
	cSquared := math.Pow(a, float64(2.0)) + math.Pow(b, float64(2.0))
	c := math.Sqrt(cSquared)

	stampSize := float64(h) / overlap
	numStamps := math.Ceil((c / stampSize))
	yFraction := a / numStamps
	xFraction := b / numStamps

	// left or right?
	xDirection := float64(1)
	if start.X > end.X {
		xDirection = float64(-1)
	}
	// up or down?
	yDirection := float64(1)
	if start.Y > end.Y {
		yDirection = float64(-1)
	}

	p := image.ZP
	x := float64(start.X)
	y := float64(start.Y)
	wHalf := w / 2
	hHalf := h / 2
	for i := 0; i < int(numStamps); i++ {

		x0 := int(math.Round(x))
		y0 := int(math.Round(y))
		r := image.Rect(x0-wHalf, y0-hHalf, x0+wHalf, y0+hHalf)
		draw.DrawMask(dst, r, color, p, rotated, p, draw.Over)

		// move along the path for the next iteration
		x += xFraction * xDirection
		y += yFraction * yDirection
	}
}

var brushCache = make(map[string]image.Image)

// loadBrushMask loads the brush stamp from the file system,
// converts it to a mask image (gray value converted to alpha channel)
// and returns an image.
func loadBrushMask(b Brush) (image.Image, error) {
	cached := brushCache[b.Name()]
	if cached != nil {
		return cached, nil
	}
	// TODO: from config
	d := "./data/brushes"
	n := b.Name() + ".png"
	p := filepath.Join(d, n)
	fmt.Printf("Load brush %q\n", p)

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	i, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	mask := createMask(i)
	brushCache[b.Name()] = mask

	return mask, nil
}

// create a mask image by using the gray value of the given image as the
// value for the mask alpha channel. Returns the mask image.
func createMask(i image.Image) image.Image {
	rect := i.Bounds()
	mask := image.NewRGBA(rect)
	x0 := rect.Min.X
	x1 := rect.Max.X
	y0 := rect.Min.Y
	y1 := rect.Max.Y

	var gray uint8
	var r, g, b uint32
	for x := x0; x < x1; x++ {
		for y := y0; y < y1; y++ {
			r, g, b, _ = i.At(x, y).RGBA()
			gray = uint8((r + g + b) / 3)
			mask.Set(x, y, color.RGBA{0, 0, 0, gray})
		}
	}

	return mask
}

// applyOpacity applies the given opacity (0.0..1.0) to the given image.
// This method returns a new image where the alpha channel is a combination
// of the source alpha and the opacity.
func applyOpacity(i image.Image, opacity float64) image.Image {
	alpha := uint8(math.Round(255 * opacity))
	mask := image.NewUniform(color.Alpha{alpha})

	rect := i.Bounds()
	dst := image.NewRGBA(rect)
	p := image.ZP
	draw.DrawMask(dst, rect, i, p, mask, p, draw.Over)
	return dst
}

// Rotate the given image counter-clockwise by angle (radians) degrees.
// Rotation is around the center of the source image.
// Returns an image with the rotated pixels.
func rotate(angle float64, i image.Image) image.Image {
	// Size of the source image
	box := i.Bounds()
	xMax := box.Max.X
	yMax := box.Max.Y

	// Create the destination image.
	// The dst size is the diagonal accross the source Rectangle.
	a := float64(box.Max.X - box.Min.X)
	b := float64(box.Max.Y - box.Min.Y)
	c := math.Sqrt(math.Pow(a, 2) + math.Pow(b, 2))
	size := int(math.Ceil(c))
	dst := image.NewRGBA(image.Rect(0, 0, size, size))

	// Rotation around center instead of origin
	// means: Translate - Rotate - Translate
	t0 := translation(-a/2, -b/2)
	rot := rotation(angle)
	t1 := translation(a/2, b/2)

	// Transform each pixel and set it on the destination image.
	var tx, ty float64
	for x := 0; x < xMax; x++ {
		for y := 0; y < yMax; y++ {
			tx, ty = float64(x), float64(y)
			tx, ty = transform(t0, tx, ty)
			tx, ty = transform(rot, tx, ty)
			tx, ty = transform(t1, tx, ty)

			tx = math.Round(tx)
			ty = math.Round(ty)
			dst.Set(int(tx), int(ty), i.At(x, y))
		}
	}

	return dst
}
