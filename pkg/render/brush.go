package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"

	"akeil.net/akeil/rm/internal/imaging"
	"akeil.net/akeil/rm/pkg/lines"
)

// see:
// https://github.com/lschwetlick/maxio/blob/master/rm_tools/rM2svg.py
// https://gitlab.com/wrobell/remt/-/blob/master/remt/drawer.py

type Brush interface {
	RenderStroke(dst draw.Image, s lines.Stroke)
}

type BasePen struct {
	mask image.Image
	fill image.Image
}

func loadBasePen(mask image.Image, c color.Color) Brush {
	return &BasePen{
		mask: mask,
		fill: image.NewUniform(c),
	}
}

func (b *BasePen) RenderStroke(dst draw.Image, s lines.Stroke) {
	walkDots(dst, s, b.renderSegment)
}

func (b *BasePen) renderSegment(dst draw.Image, start, end lines.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(b.mask, width, opacity, start, end)
	overlap := 2.0
	drawStamp(dst, mask, b.fill, start, end, overlap)
}

// Ballpoint ------------------------------------------------------------------

// The Ballpoint pen has some sensitivity for pressure
type Ballpoint struct {
	mask  image.Image
	fill  image.Image
	color color.Color
}

func (b *Ballpoint) RenderStroke(dst draw.Image, s lines.Stroke) {
	drawPath(dst, s, b.color)
}

// Fineliner ------------------------------------------------------------------

// Fineliner has no sensitivity to pressure or tilt.
type Fineliner struct {
	mask  image.Image
	fill  image.Image
	color color.Color
}

func (f *Fineliner) RenderStroke(dst draw.Image, s lines.Stroke) {
	drawPath(dst, s, f.color)
}

// Pencil ---------------------------------------------------------------------

type Pencil struct {
	mask image.Image
	fill image.Image
}

func (p *Pencil) RenderStroke(dst draw.Image, s lines.Stroke) {
	walkDots(dst, s, p.renderSegment)
}

func (p *Pencil) renderSegment(dst draw.Image, start, end lines.Dot) {
	// TODO: pencil rendering does not look good
	// *desity* of pixels in stamp should vary with tilt and pressure - how to to it?

	width := float64(start.Width)

	// pencil has high sensitivity to pressure
	x := math.Pow(float64(start.Pressure), 4)
	y := 0.1
	opacity := x*y + 1 - y

	mask := prepareMask(p.mask, width, opacity, start, end)
	overlap := 1.5
	drawStamp(dst, mask, p.fill, start, end, overlap)
}

// Mechanical Pencil ----------------------------------------------------------

type MechanicalPencil struct {
	mask image.Image
	fill image.Image
}

func (m *MechanicalPencil) RenderStroke(dst draw.Image, s lines.Stroke) {
	walkDots(dst, s, m.renderSegment)
}

func (m *MechanicalPencil) renderSegment(dst draw.Image, start, end lines.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(m.mask, width, opacity, start, end)
	overlap := 4.0
	drawStamp(dst, mask, m.fill, start, end, overlap)
}

// Marker ---------------------------------------------------------------------

type Marker struct {
	mask image.Image
	fill image.Image
}

func (m *Marker) RenderStroke(dst draw.Image, s lines.Stroke) {
	walkDots(dst, s, m.renderSegment)
}

func (m *Marker) renderSegment(dst draw.Image, start, end lines.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(m.mask, width, opacity, start, end)
	overlap := 4.0
	drawStamp(dst, mask, m.fill, start, end, overlap)
}

// Highlighter ----------------------------------------------------------------

type Highlighter struct {
	mask image.Image
	fill image.Image
}

func (h *Highlighter) RenderStroke(dst draw.Image, s lines.Stroke) {
	// TODO - poor performance? -> Needs measure.

	// The highlighter has a uniform opacity per stroke.
	// This means overlapping segments do not add up their opacity values.

	// To achieve this, render all segments in full opacity on a temp image...
	rect := dst.Bounds()
	tmp := image.NewRGBA(rect)
	walkDots(tmp, s, h.renderSegment)

	// ... then transfer the temp image with desired opacity onto the actual
	// destination.
	opacity := 0.4
	alpha := uint8(math.Round(255 * opacity))
	mask := image.NewUniform(color.Alpha{alpha})

	draw.DrawMask(dst, rect, tmp, image.Point{}, mask, image.Point{}, draw.Over)
}

func (h *Highlighter) renderSegment(dst draw.Image, start, end lines.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(h.mask, width, opacity, start, end)
	overlap := 1.0
	drawStamp(dst, mask, h.fill, start, end, overlap)
}

// Paintbrush -----------------------------------------------------------------

type Paintbrush struct {
	fill color.Color
}

func (p *Paintbrush) RenderStroke(dst draw.Image, s lines.Stroke) {
	// TODO: probably better with a stamp image
	drawPath(dst, s, p.fill)
}

// Rendering Helpers ----------------------------------------------------------

type segmentRenderer func(dst draw.Image, start, end lines.Dot)

func walkDots(dst draw.Image, s lines.Stroke, render segmentRenderer) {
	for i := 1; i < len(s.Dots); i++ {
		start := s.Dots[i-1]
		end := s.Dots[i]
		render(dst, start, end)
	}
}

// Prepare the mask image by scaling it to the desired width, applying opacity.
// and rotating it to align with the segment from start to end.
func prepareMask(mask image.Image, width, opacity float64, start, end lines.Dot) image.Image {
	i := imaging.Resize(mask, width)

	if opacity != 1.0 {
		i = imaging.ApplyOpacity(i, opacity)
	}

	// Rotate the brush to align with the path.
	// Brush images are alinged "left to right", i.e. the "front" is on the left.
	angle := math.Atan2(float64(start.Y-end.Y), float64(start.X-end.X))
	return imaging.Rotate(angle, i)
}

// Draw a single line from start to end with a "stamp" image.
// The stamp image is repeated along the line, taking the overlap factor into account.
func drawStamp(dst draw.Image, mask image.Image, fill image.Image, start, end lines.Dot, overlap float64) {
	rect := mask.Bounds()
	w := rect.Max.X - rect.Min.X
	h := rect.Max.Y - rect.Min.Y

	// calculate the length of the segment
	a := math.Abs(float64(start.Y - end.Y))
	b := math.Abs(float64(start.X - end.X))
	cSquared := math.Pow(a, float64(2.0)) + math.Pow(b, float64(2.0))
	length := math.Sqrt(cSquared)

	stampSize := float64(h) / overlap // assumes stamps are quadratic
	numStamps := math.Ceil((length / stampSize))
	yFraction := a / numStamps
	xFraction := b / numStamps

	// left or right?
	xDirection := float64(1)
	if start.X > end.X {
		xDirection *= -1
	}
	// up or down?
	yDirection := float64(1)
	if start.Y > end.Y {
		yDirection *= -1
	}

	x := float64(start.X)
	y := float64(start.Y)
	wHalf := w / 2
	hHalf := h / 2
	for i := 0; i < int(numStamps); i++ {

		x0 := int(math.Round(x))
		y0 := int(math.Round(y))
		r := image.Rect(x0-wHalf, y0-hHalf, x0+wHalf, y0+hHalf)
		draw.DrawMask(dst, r, fill, image.Point{}, mask, image.Point{}, draw.Over)

		// move along the path for the next iteration
		x += xFraction * xDirection
		y += yFraction * yDirection
	}
}

// Draw the given stroke with basic draw2d path functions.
// This works well for brushes with little variance in line width
// and which do not have the need for texture.
func drawPath(dst draw.Image, s lines.Stroke, c color.Color) {
	// guard - we'll access by index later
	if len(s.Dots) == 0 {
		return
	}

	gc := draw2dimg.NewGraphicContext(dst)
	defer gc.Close()

	gc.SetStrokeColor(c)
	gc.SetLineCap(draw2d.RoundCap)
	gc.SetLineJoin(draw2d.RoundJoin)

	d := s.Dots[0]
	x := float64(d.X)
	y := float64(d.Y)
	w := float64(d.Width)
	gc.BeginPath()
	gc.SetLineWidth(w)
	gc.MoveTo(x, y)

	// Remove precision from float values
	coarse := func(v float64) float64 {
		return math.Round(v*10) / 10
	}
	// We'll close and stroke sub-segments of the stroke whenver the width changes.
	// For this, we need to remember position and width of the previous dot.
	xPrev := x
	yPrev := x
	wPrev := w
	points := 0

	// starts with the *second* dot
	for i := 1; i < len(s.Dots); i++ {
		d = s.Dots[i]
		x = float64(d.X)
		y = float64(d.Y)
		w := float64(d.Width)

		// We cannot stroke paths with variable width.
		// So everytime width changes, stroke the current path
		// and start a new one with the changed width.
		if coarse(w) != coarse(wPrev) && points > 0 {
			gc.Stroke()

			gc.BeginPath()
			gc.SetLineWidth(w)
			gc.MoveTo(xPrev, yPrev)
			points = 0
		}

		gc.LineTo(x, y)
		points++

		xPrev = x
		yPrev = y
		wPrev = w
	}

	if points > 0 {
		gc.Stroke()
	}
}
