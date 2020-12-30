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
	"sync"

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

// renderStroke paints a single stroke on the destination image.
func renderStroke(dst draw.Image, s rm.Stroke) error {
	col := colors[s.BrushColor]
	if col == nil {
		return fmt.Errorf("invalid color %v", s.BrushColor)
	}

	pen, err := loadBrush(s.BrushType, col)
	if err != nil {
		return err
	}

	pen.RenderStroke(dst, s)
	return nil
}

var cache = make(map[string]image.Image)
var cacheMx sync.Mutex

func readPNG(subdir, name string) (image.Image, error) {
	cacheMx.Lock()
	defer cacheMx.Unlock()

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
