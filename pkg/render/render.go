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
	"akeil.net/akeil/rm/internal/imaging"
	"akeil.net/akeil/rm/internal/logging"
)

var colors = map[rm.BrushColor]color.Color{
	rm.Black: color.Black,
	rm.Gray:  color.RGBA{127, 127, 127, 255},
	rm.White: color.White,
}
var bgColor = color.White

// Drawing paints the given drawing and writes the result to the given
// writer.
func Drawing(d *rm.Drawing, w io.Writer) error {
	err := renderPNG(d, true, w)
	if err != nil {
		return err
	}

	return nil
}

// Page renders the page from the given document and writes the
// result to the given writer.
//
// Unlike RenderDrawing, this includes the page's background template.
func Page(doc *rm.Document, pageID string, w io.Writer) error {
	p, err := doc.Page(pageID)
	if err != nil {
		return err
	}

	d, err := doc.Drawing(pageID)
	if err != nil {
		return err
	}

	r := image.Rect(0, 0, rm.MaxWidth, rm.MaxHeight)
	dst := image.NewRGBA(r)

	if p.HasTemplate() {
		err = renderTemplate(dst, p.Template(), p.Orientation())
		if err != nil {
			return err
		}
	}

	err = renderLayers(dst, d)
	if err != nil {
		return err
	}

	// Now that we are done with transparency...
	grayscale := imaging.ToGray(dst)

	err = png.Encode(w, grayscale)
	if err != nil {
		return err
	}

	return nil
}

// RenderPNG paints the given drawing to a PNG file and writes the PNG data
// to the given writer.
func renderPNG(d *rm.Drawing, bg bool, w io.Writer) error {
	r := image.Rect(0, 0, rm.MaxWidth, rm.MaxHeight)
	dst := image.NewRGBA(r)

	if bg {
		renderBackground(dst)
	}

	err := renderLayers(dst, d)
	if err != nil {
		return err
	}

	err = png.Encode(w, dst)
	if err != nil {
		return err
	}

	return nil
}

func renderLayers(dst draw.Image, d *rm.Drawing) error {
	for _, l := range d.Layers {
		err := renderLayer(dst, l)
		if err != nil {
			return err
		}
	}
	return nil
}

func renderTemplate(dst draw.Image, tpl string, layout rm.Orientation) error {
	i, err := readPNG("templates", tpl)
	if err != nil {
		return err
	}

	if layout == rm.Landscape {
		i = imaging.Rotate(rad(90), i)
	}

	p := image.ZP
	draw.Draw(dst, dst.Bounds(), i, p, draw.Over)

	return nil
}

// renderBackground fills the complete destination image with the background color (white).
func renderBackground(dst draw.Image) {
	bg := image.NewUniform(bgColor)
	p := image.ZP
	draw.Draw(dst, dst.Bounds(), bg, p, draw.Over)
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
	scaled := imaging.Resize(mask, scale)

	// Apply additional opacity for pressure/speed
	opacity := pen.Opacity(start.Pressure, start.Speed)
	opaque := imaging.ApplyOpacity(scaled, opacity)

	// Rotate the brush to align with the path
	angle := math.Atan2(float64(start.Y-end.Y), float64(start.X-end.X))
	rotated := imaging.Rotate(angle, opaque)

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

// loadBrushMask loads the brush stamp from the file system,
// converts it to a mask image (gray value converted to alpha channel)
// and returns an image.
func loadBrushMask(b Brush) (image.Image, error) {
	i, err := readPNG("brushes", b.Name())
	if err != nil {
		return nil, err
	}

	mask := imaging.CreateMask(i)

	return mask, nil
}

var cache = make(map[string]image.Image)

func readPNG(subdir, name string) (image.Image, error) {
	key := subdir + "/" + name
	cached := cache[key]
	if cached != nil {
		return cached, nil
	}

	// TODO: data-dir from config
	d := "./data"
	n := name + ".png"
	p := filepath.Join(d, subdir, n)
	logging.Debug("Load PNG %q\n", p)

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

	cache[key] = i

	return i, nil
}

func rad(deg float64) float64 {
	return deg * (math.Pi / 180)
}
