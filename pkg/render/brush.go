package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/imaging"
)

// see:
// https://github.com/lschwetlick/maxio/blob/master/rm_tools/rM2svg.py

type Brush interface {
	Name() string
	Opacity(pressure, speed float32) float64
	Width(base, pressure, tilt float32) float64
	Overlap() float64
	RenderSegment(dst draw.Image, start, end rm.Dot)
}

func NewBrush(t rm.BrushType, c color.Color) (Brush, error) {
	switch t {
	case rm.Ballpoint, rm.BallpointV5:
		return loadBallpoint(c)
	case rm.Pencil, rm.PencilV5:
		return loadPencil(c)
	case rm.MechanicalPencil, rm.MechanicalPencilV5:
		return loadMechanicalPencil(c)
	case rm.Marker, rm.MarkerV5:
		return loadMarker(c)
	case rm.Fineliner, rm.FinelinerV5:
		return loadFineliner(c)
	case rm.Highlighter, rm.HighlighterV5:
		return loadHighlighter(c)
	default:
		return loadBasePen(c)
	}
}

type BasePen struct {
	mask image.Image
	fill image.Image
}

func loadBasePen(c color.Color) (Brush, error) {
	i, err := readPNG("brushes", "ballpoint")
	if err != nil {
		return nil, err
	}

	return &BasePen{
		mask: imaging.CreateMask(i),
		fill: image.NewUniform(c),
	}, nil
}

func (b *BasePen) Name() string {
	return "minus"
}

func (b *BasePen) Opacity(pressure, speed float32) float64 {
	return 1.0
}

func (b *BasePen) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (b *BasePen) Overlap() float64 {
	return 1.0
}

func (b *BasePen) RenderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(b.mask, width, opacity, start, end)
	overlap := 1.0
	drawPath(dst, mask, b.fill, start, end, overlap)
}

// Ballpoint ------------------------------------------------------------------

// The Ballpoint pen has some sensitivity for pressure
type Ballpoint struct {
	mask image.Image
	fill image.Image
}

func loadBallpoint(c color.Color) (Brush, error) {
	i, err := readPNG("brushes", "ballpoint")
	if err != nil {
		return nil, err
	}

	return &Ballpoint{
		mask: imaging.CreateMask(i),
		fill: image.NewUniform(c),
	}, nil
}

func (b *Ballpoint) Name() string {
	return "ballpoint"
}

func (b *Ballpoint) Opacity(pressure, speed float32) float64 {
	// giving some opacity makes linkes look smaller
	x := math.Pow(float64(pressure), 2)
	y := 0.2
	return (x * y) + (1.0 - y)
}

func (b *Ballpoint) Width(base, pressure, tilt float32) float64 {
	w := float64(base)

	// make sure lines have a minimum width
	// TODO: tke BrushSize into account
	minWidth := 3.0
	w = math.Max(w, minWidth)

	// high pressure lines are a little bit wider
	x := math.Pow(float64(pressure), 2)
	y := 0.3

	return w + w*y*x
}

func (b *Ballpoint) Overlap() float64 {
	return 2.0
}

func (b *Ballpoint) RenderSegment(dst draw.Image, start, end rm.Dot) {
	width := b.Width(start.Width, start.Pressure, start.Tilt)
	opacity := b.Opacity(start.Pressure, start.Speed)
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

func loadFineliner(c color.Color) (Brush, error) {
	i, err := readPNG("brushes", "fineliner")
	if err != nil {
		return nil, err
	}

	return &Fineliner{
		mask: imaging.CreateMask(i),
		fill: image.NewUniform(c),
	}, nil
}

func (f *Fineliner) Name() string {
	return "fineliner"
}

func (f *Fineliner) Opacity(pressure, speed float32) float64 {
	return 1.0
}

func (b *Fineliner) Width(base, pressure, tilt float32) float64 {
	mindWidth := 3.0
	return math.Max(float64(base), mindWidth)
}

func (f *Fineliner) Overlap() float64 {
	return 3.0
}

func (f *Fineliner) RenderSegment(dst draw.Image, start, end rm.Dot) {
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

func loadPencil(c color.Color) (Brush, error) {
	i, err := readPNG("brushes", "pencil")
	if err != nil {
		return nil, err
	}

	return &Pencil{
		mask: imaging.CreateMask(i),
		fill: image.NewUniform(c),
	}, nil
}

func (p *Pencil) Name() string {
	return "pencil"
}

func (p *Pencil) Opacity(pressure, speed float32) float64 {
	// pencil has high sensitivity to pressure
	x := math.Pow(float64(pressure), 4)
	y := 0.1
	return x*y + 1 - y
	//return 1.0
}

func (p *Pencil) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (p *Pencil) Overlap() float64 {
	return 1.5
}

func (p *Pencil) RenderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := p.Opacity(start.Pressure, start.Speed)
	mask := prepareMask(p.mask, width, opacity, start, end)
	overlap := 1.5
	drawPath(dst, mask, p.fill, start, end, overlap)
}

// Mechanical Pencil ----------------------------------------------------------

type MechanicalPencil struct {
	mask image.Image
	fill image.Image
}

func loadMechanicalPencil(c color.Color) (Brush, error) {
	i, err := readPNG("brushes", "mech-pencil")
	if err != nil {
		return nil, err
	}

	return &MechanicalPencil{
		mask: imaging.CreateMask(i),
		fill: image.NewUniform(c),
	}, nil
}

func (m *MechanicalPencil) Name() string {
	return "mech-pencil"
}

func (m *MechanicalPencil) Opacity(pressure, speed float32) float64 {
	// pencil has medium sensitivity to pressure
	//x := math.Pow(float64(pressure), 4)
	//return x*0.8 + 0.2
	return 1.0
}

func (m *MechanicalPencil) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (m *MechanicalPencil) Overlap() float64 {
	return 4.0
}

func (m *MechanicalPencil) RenderSegment(dst draw.Image, start, end rm.Dot) {
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

func loadMarker(c color.Color) (Brush, error) {
	i, err := readPNG("brushes", "marker")
	if err != nil {
		return nil, err
	}

	return &Marker{
		mask: imaging.CreateMask(i),
		fill: image.NewUniform(c),
	}, nil
}

func (m *Marker) Name() string {
	return "marker"
}

func (m *Marker) Opacity(pressure, speed float32) float64 {
	return 1.0
}

func (m *Marker) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (m *Marker) Overlap() float64 {
	return 6.0
}

func (m *Marker) RenderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 1.0
	mask := prepareMask(m.mask, width, opacity, start, end)
	overlap := 6.0
	drawPath(dst, mask, m.fill, start, end, overlap)
}

// Highlighter ----------------------------------------------------------------

type Highlighter struct {
	mask image.Image
	fill image.Image
}

func loadHighlighter(c color.Color) (Brush, error) {
	i, err := readPNG("brushes", "highlighter")
	if err != nil {
		return nil, err
	}

	return &Highlighter{
		mask: imaging.CreateMask(i),
		fill: image.NewUniform(c),
	}, nil
}

func (h *Highlighter) Name() string {
	return "highlighter"
}

func (h *Highlighter) Opacity(pressure, speed float32) float64 {
	// marker has no sensitivity to pressure
	return 0.1
}

func (b *Highlighter) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (h *Highlighter) Overlap() float64 {
	return 3.0
}

func (h *Highlighter) RenderSegment(dst draw.Image, start, end rm.Dot) {
	width := float64(start.Width)
	opacity := 0.1
	mask := prepareMask(h.mask, width, opacity, start, end)
	overlap := 3.0
	drawPath(dst, mask, h.fill, start, end, overlap)
}

func prepareMask(mask image.Image, width, opacity float64, start, end rm.Dot) image.Image {
	i := imaging.Resize(mask, width)

	if opacity != 1.0 {
		i = imaging.ApplyOpacity(i, opacity)
	}

	// Rotate the brush to align with the path
	angle := math.Atan2(float64(start.Y-end.Y), float64(start.X-end.X))
	return imaging.Rotate(angle, i)
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
