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
	Overlap() float64
}

func NewBrush(t rm.BrushType) Brush {
	//fmt.Printf("Brush: size=%v\n", s)
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

func (b *BasePen) Overlap() float64 {
	return 1.0
}

// Ballpoint ------------------------------------------------------------------

type Ballpoint struct {
}

func (b *Ballpoint) Name() string {
	return "ballpoint"
}

func (b *Ballpoint) Opacity(pressure, speed float32) float64 {
	// ballpoint has low sensitivity for pressure
	x := math.Pow(float64(pressure), 4)
	return x*0.25 + 0.75
}
func (b *Ballpoint) Overlap() float64 {
	return 4.0
}

// Fineliner ------------------------------------------------------------------

type Fineliner struct{}

func (f *Fineliner) Name() string {
	return "fineliner"
}

func (f *Fineliner) Opacity(pressure, speed float32) float64 {
	return 1.0
}

func (f *Fineliner) Overlap() float64 {
	return 4.0
}

// Pencil ---------------------------------------------------------------------

type Pencil struct {
}

func (p *Pencil) Name() string {
	return "pencil-2"
}

func (p *Pencil) Opacity(pressure, speed float32) float64 {
	// pencil has high sensitivity to pressure
	x := math.Pow(float64(pressure), 4)
	return x*0.9 + 0.1
}

func (p *Pencil) Overlap() float64 {
	return 3.0
}

// Mechanical Pencil ----------------------------------------------------------

type MechanicalPencil struct{}

func (m *MechanicalPencil) Name() string {
	return "mech-pencil"
}

func (m *MechanicalPencil) Opacity(pressure, speed float32) float64 {
	// pencil has medium sensitivity to pressure
	x := math.Pow(float64(pressure), 4)
	return x*0.8 + 0.2
}

func (m *MechanicalPencil) Overlap() float64 {
	return 2.0
}

// Marker ---------------------------------------------------------------------

type Marker struct {
}

func (m *Marker) Name() string {
	return "marker"
}

func (m *Marker) Opacity(pressure, speed float32) float64 {
	// marker has almost no sensitivity to pressure
	x := math.Pow(float64(pressure), 2)
	return x + 0.9
}

func (m *Marker) Overlap() float64 {
	return 2.0
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

func (h *Highlighter) Overlap() float64 {
	return 3.0
}
