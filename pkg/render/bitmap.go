package render

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/imaging"
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
		err = renderTemplate(c, dst, p.Template(), p.Orientation())
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

func renderTemplate(c *Context, dst draw.Image, tpl string, layout rm.Orientation) error {
	i, err := c.loadTemplate(tpl)
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

func rad(deg float64) float64 {
	return deg * (math.Pi / 180)
}
