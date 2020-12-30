package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/llgcode/draw2d/draw2dimg"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/imaging"
)

// see:
// https://github.com/lschwetlick/maxio/blob/master/rm_tools/rM2svg.py

type Brush interface {
	RenderStroke(dst draw.Image, s rm.Stroke)
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

func (b *BasePen) RenderStroke(dst draw.Image, s rm.Stroke) {
	walkDots(dst, s, b.renderSegment)
}

func (b *BasePen) renderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(b.mask, width, opacity, start, end)
	overlap := 2.0
	drawPath(dst, mask, b.fill, start, end, overlap)
}

// Ballpoint ------------------------------------------------------------------

// The Ballpoint pen has some sensitivity for pressure
type Ballpoint struct {
	mask image.Image
	fill image.Image
}

func (b *Ballpoint) RenderStroke(dst draw.Image, s rm.Stroke) {
	walkDots(dst, s, b.renderSegment)
}

func (b *Ballpoint) renderSegment(dst draw.Image, start, end rm.Dot) {
	// make sure lines have a minimum width
	// TODO: tke BrushSize into account
	minWidth := 3.0
	w := math.Max(float64(start.Width), minWidth)
	// high pressure lines are a little bit wider
	x := math.Pow(float64(start.Pressure), 2)
	y := 0.3
	width := w + w*y*x

	k := math.Pow(float64(start.Pressure), 2)
	l := 0.2
	opacity := (k * l) + (1.0 - l)

	mask := prepareMask(b.mask, width, opacity, start, end)
	overlap := 2.0
	drawPath(dst, mask, b.fill, start, end, overlap)
}

// Fineliner ------------------------------------------------------------------

// Fineliner has no sensitivity to pressure or tilt.
type Fineliner struct {
	mask image.Image
	fill image.Image
}

func (f *Fineliner) RenderStroke(dst draw.Image, s rm.Stroke) {
	walkDots(dst, s, f.renderSegment)
}

func (f *Fineliner) renderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(f.mask, width, opacity, start, end)
	overlap := 3.0
	drawPath(dst, mask, f.fill, start, end, overlap)
}

// Pencil ---------------------------------------------------------------------

type Pencil struct {
	mask image.Image
	fill image.Image
}

func (p *Pencil) RenderStroke(dst draw.Image, s rm.Stroke) {
	walkDots(dst, s, p.renderSegment)
}

func (p *Pencil) renderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)

	// pencil has high sensitivity to pressure
	x := math.Pow(float64(start.Pressure), 4)
	y := 0.1
	opacity := x*y + 1 - y

	mask := prepareMask(p.mask, width, opacity, start, end)
	overlap := 1.5
	drawPath(dst, mask, p.fill, start, end, overlap)
}

// Mechanical Pencil ----------------------------------------------------------

type MechanicalPencil struct {
	mask image.Image
	fill image.Image
}

func (m *MechanicalPencil) RenderStroke(dst draw.Image, s rm.Stroke) {
	walkDots(dst, s, m.renderSegment)
}

func (m *MechanicalPencil) renderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(m.mask, width, opacity, start, end)
	overlap := 4.0
	drawPath(dst, mask, m.fill, start, end, overlap)
}

// Marker ---------------------------------------------------------------------

type Marker struct {
	mask image.Image
	fill image.Image
}

func (m *Marker) RenderStroke(dst draw.Image, s rm.Stroke) {
	walkDots(dst, s, m.renderSegment)
}

func (m *Marker) renderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(m.mask, width, opacity, start, end)
	overlap := 4.0
	drawPath(dst, mask, m.fill, start, end, overlap)
}

// Highlighter ----------------------------------------------------------------

type Highlighter struct {
	mask image.Image
	fill image.Image
}

func (h *Highlighter) RenderStroke(dst draw.Image, s rm.Stroke) {
	// The highlighter has a uniform opacity per stroke.
	// This means overlapping segments do not add up their opacity values.

	// To achieve this, render all segments in full opacity on a temp image...
	r := dst.Bounds()
	tmp := image.NewRGBA(r)
	walkDots(tmp, s, h.renderSegment)

	// ... then transfer the temp image with desired opacity onto the actual
	// destination.
	opacity := 0.4
	a := uint8(math.Round(255 * opacity))
	mask := image.NewUniform(color.Alpha{a})
	p := image.ZP
	draw.DrawMask(dst, r, tmp, p, mask, p, draw.Over)
}

func (h *Highlighter) renderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(h.mask, width, opacity, start, end)
	overlap := 1.0
	drawPath(dst, mask, h.fill, start, end, overlap)
}

// Paintbrush -----------------------------------------------------------------

type Paintbrush struct {
	fill color.Color
}

func (p *Paintbrush) RenderStroke(dst draw.Image, s rm.Stroke) {
	walkDots(dst, s, p.renderSegment)
}

func (p *Paintbrush) renderSegment(dst draw.Image, start, end rm.Dot) {
	gc := draw2dimg.NewGraphicContext(dst)

	gc.SetStrokeColor(p.fill)
	gc.SetLineWidth(float64(start.Width))

	gc.BeginPath()
	gc.MoveTo(float64(start.X), float64(start.Y))
	gc.LineTo(float64(end.X), float64(end.Y))
	gc.Stroke()
}

// Rendering Helpers ----------------------------------------------------------

type segmentRenderer func(dst draw.Image, start, end rm.Dot)

func walkDots(dst draw.Image, s rm.Stroke, r segmentRenderer) {
	for i := 1; i < len(s.Dots); i++ {
		start := s.Dots[i-1]
		end := s.Dots[i]
		r(dst, start, end)
	}
}

func prepareMask(mask image.Image, width, opacity float64, start, end rm.Dot) image.Image {
	i := imaging.Resize(mask, width)
	if opacity != 1.0 {
		i = imaging.ApplyOpacity(i, opacity)
	}

	// Rotate the brush to align with the path.
	// Brush images are alinged "left to right", i.e. the "front" is on the left.
	angle := math.Atan2(float64(start.Y-end.Y), float64(start.X-end.X))
	return imaging.Rotate(angle, i)
	return i
}

func drawPath(dst draw.Image, mask image.Image, fill image.Image, start, end rm.Dot, overlap float64) {
	r := mask.Bounds()
	w, h := r.Max.X-r.Min.X, r.Max.Y-r.Min.Y

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
		draw.DrawMask(dst, r, fill, p, mask, p, draw.Over)

		// move along the path for the next iteration
		x += xFraction * xDirection
		y += yFraction * yDirection
	}
}
