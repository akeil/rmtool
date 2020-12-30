package render

import (
	"bufio"
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

var bgColor = color.White

// Page renders the page from the given document and writes the
// result to the given writer.
//
// Unlike RenderDrawing, this includes the page's background template.
func Page(doc *rm.Document, pageID string, w io.Writer) error {
	r := NewContext("./data")
	return renderPage(r, doc, pageID, w)
}

func renderPage(c *Context, doc *rm.Document, pageID string, w io.Writer) error {
	p, err := doc.Page(pageID)
	if err != nil {
		return err
	}

	d, err := doc.Drawing(pageID)
	if err != nil {
		return err
	}

	rect := image.Rect(0, 0, rm.MaxWidth, rm.MaxHeight)
	dst := image.NewRGBA(rect)

	if p.HasTemplate() {
		err = renderTemplate(dst, p.Template(), p.Orientation())
		if err != nil {
			return err
		}
	}

	err = renderLayers(c, dst, d)
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
func renderPNG(c *Context, d *rm.Drawing, bg bool, w io.Writer) error {
	r := image.Rect(0, 0, rm.MaxWidth, rm.MaxHeight)
	dst := image.NewRGBA(r)

	if bg {
		renderBackground(dst)
	}

	err := renderLayers(c, dst, d)
	if err != nil {
		return err
	}

	err = png.Encode(w, dst)
	if err != nil {
		return err
	}

	return nil
}

func renderLayers(c *Context, dst draw.Image, d *rm.Drawing) error {
	for _, l := range d.Layers {
		err := renderLayer(c, dst, l)
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
func renderLayer(c *Context, dst draw.Image, l rm.Layer) error {
	for _, s := range l.Strokes {
		// The erased content is deleted,
		// but eraser strokes are recorded.
		if s.BrushType == rm.Eraser {
			continue
		}

		err := renderStroke(c, dst, s)
		if err != nil {
			return err
		}
	}

	return nil
}

// renderStroke paints a single stroke on the destination image.
func renderStroke(c *Context, dst draw.Image, s rm.Stroke) error {
	brush, err := c.loadBrush(s.BrushType, s.BrushColor)
	if err != nil {
		return err
	}

	brush.RenderStroke(dst, s)
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
