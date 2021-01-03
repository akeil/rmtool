package render

import (
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"

	"github.com/akeil/rmtool"
	"github.com/akeil/rmtool/internal/imaging"
	"github.com/akeil/rmtool/pkg/lines"
)

// Page renders the page from the given document and writes the
// result to the given writer.
//
// Unlike RenderDrawing, this includes the page's background template.
func Page(doc *rmtool.Document, pageID string, w io.Writer) error {
	c := DefaultContext()
	return renderPage(c, doc, pageID, w)
}

func renderPage(c *Context, doc *rmtool.Document, pageID string, w io.Writer) error {
	pg, err := doc.Page(pageID)
	if err != nil {
		return err
	}

	rect := image.Rect(0, 0, lines.MaxWidth, lines.MaxHeight)
	dst := image.NewRGBA(rect)

	if pg.HasTemplate() {
		err = renderTemplate(c, dst, pg.Template(), doc.Orientation())
		if err != nil {
			return err
		}
	}

	d, err := doc.Drawing(pageID)
	if err != nil {
		return err
	}

	err = renderLayers(c, dst, d)
	if err != nil {
		return err
	}

	return png.Encode(w, dst)
}

// RenderPNG paints the given drawing to a PNG file and writes the PNG data
// to the given writer.
func renderPNG(c *Context, d *lines.Drawing, paintBg bool, w io.Writer) error {
	rect := image.Rect(0, 0, lines.MaxWidth, lines.MaxHeight)
	dst := image.NewRGBA(rect)

	if paintBg {
		renderBackground(c, dst)
	}

	err := renderLayers(c, dst, d)
	if err != nil {
		return err
	}

	return png.Encode(w, dst)
}

// renderTemplate paints the named background template on the given destination
// image.
//
// The background image is loaded from the given Context.
//
// An error is returned ff the template cannot be loaded.
func renderTemplate(c *Context, dst draw.Image, tpl string, layout rmtool.Orientation) error {
	img, err := c.loadTemplate(tpl)
	if err != nil {
		return err
	}

	if layout == rmtool.Landscape {
		img = imaging.Rotate(rad(90), img)
	}

	draw.Draw(dst, dst.Bounds(), img, image.Point{}, draw.Over)

	return nil
}

// renderBackground fills the complete destination image with the background color (white).
func renderBackground(c *Context, dst draw.Image) {
	bg := image.NewUniform(c.palette.Background)
	draw.Draw(dst, dst.Bounds(), bg, image.Point{}, draw.Over)
}

// renderLayoers paints all layers on the destination image.
func renderLayers(c *Context, dst draw.Image, d *lines.Drawing) error {
	for _, l := range d.Layers {
		for _, s := range l.Strokes {
			// The erased content is deleted,
			// but eraser strokes are recorded.
			if s.BrushType == lines.Eraser || s.BrushType == lines.EraseArea {
				continue
			}

			brush, err := c.loadBrush(s.BrushType, s.BrushColor)
			if err != nil {
				return err
			}

			brush.RenderStroke(dst, s)
		}
	}

	return nil
}

func rad(deg float64) float64 {
	return deg * (math.Pi / 180)
}
