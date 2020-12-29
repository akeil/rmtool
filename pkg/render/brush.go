package render

import (
	"math"

	"akeil.net/akeil/rm"
)

// see:
// https://github.com/lschwetlick/maxio/blob/master/rm_tools/rM2svg.py

type Brush interface {
	Name() string
	Opacity(pressure, speed float32) float64
	Width(base, pressure, tilt float32) float64
	Overlap() float64
}

func NewBrush(t rm.BrushType) Brush {
	switch t {
	case rm.Ballpoint, rm.BallpointV5:
		return &Ballpoint{}
	case rm.Pencil, rm.PencilV5:
		return &Pencil{}
	case rm.MechanicalPencil, rm.MechanicalPencilV5:
		return &MechanicalPencil{}
	case rm.Marker, rm.MarkerV5:
		return &Marker{}
	case rm.Fineliner, rm.FinelinerV5:
		return &Fineliner{}
	case rm.Highlighter, rm.HighlighterV5:
		return &Highlighter{}
	default:
		return &BasePen{}
	}
}

type BasePen struct {
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

// Ballpoint ------------------------------------------------------------------

// The Ballpoint pen has some sensitivity for pressure
type Ballpoint struct {
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
	minWidth := 3.5
	w = math.Max(w, minWidth)

	// high pressure lines are a little bit wider
	x := math.Pow(float64(pressure), 2)
	y := 0.3

	return w + w*y*x
}

func (b *Ballpoint) Overlap() float64 {
	return 2.0
}

// Fineliner ------------------------------------------------------------------

// Fineliner has no sensitivity to pressure or tilt.
type Fineliner struct{}

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

// Pencil ---------------------------------------------------------------------

type Pencil struct {
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

func (b *Pencil) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (p *Pencil) Overlap() float64 {
	return 1.5
}

// Mechanical Pencil ----------------------------------------------------------

type MechanicalPencil struct{}

func (m *MechanicalPencil) Name() string {
	return "mech-pencil"
}

func (m *MechanicalPencil) Opacity(pressure, speed float32) float64 {
	// pencil has medium sensitivity to pressure
	//x := math.Pow(float64(pressure), 4)
	//return x*0.8 + 0.2
	return 1.0
}

func (b *MechanicalPencil) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (m *MechanicalPencil) Overlap() float64 {
	return 4.0
}

// Marker ---------------------------------------------------------------------

type Marker struct {
}

func (m *Marker) Name() string {
	return "marker"
}

func (m *Marker) Opacity(pressure, speed float32) float64 {
	return 1.0
}

func (b *Marker) Width(base, pressure, tilt float32) float64 {
	return float64(base)
}

func (m *Marker) Overlap() float64 {
	return 6.0
}

// Highlighter ----------------------------------------------------------------

type Highlighter struct{}

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
