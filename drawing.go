package rmtool

// Version defines the version number of a remarkable note.
type Version int

const (
	V3 Version = iota
	V5
)

// BrushColor defines the color of the brush (black, gray, white).
type BrushColor uint32

const (
	Black BrushColor = 0
	Gray  BrushColor = 1
	White BrushColor = 2
)

// BrushType is one of the predefined brush types.
type BrushType uint32

const (
	// v3 types
	PaintBrush       BrushType = 0
	Pencil           BrushType = 1
	Ballpoint        BrushType = 2
	Marker           BrushType = 3
	Fineliner        BrushType = 4
	Highlighter      BrushType = 5
	Eraser           BrushType = 6
	MechanicalPencil BrushType = 7
	EraseArea        BrushType = 8

	// v5 types
	BrushV5            BrushType = 12
	MechanicalPencilV5 BrushType = 13
	PencilV5           BrushType = 14
	BallpointV5        BrushType = 15
	MarkerV5           BrushType = 16
	FinelinerV5        BrushType = 17
	HighlighterV5      BrushType = 18
	//CalligraphyV5 = ?
)

// BrushSize represents the base brush sizes.
type BrushSize float32

// These are the three sizes available in the UI.
// Other values are possible, e.g. through scaling.
const (
	Small  BrushSize = 1.875
	Medium BrushSize = 2.0
	Large  BrushSize = 2.125
)

// Drawing represents a single page with drawings.
type Drawing struct {
	Version Version
	Layers  []Layer
}

// NewDrawing creates a new page.
func newDrawing() *Drawing {
	return &Drawing{Version: V5}
}

// NumLayers returns the number of layers in the drawing.
func (d *Drawing) NumLayers() int {
	return len(d.Layers)
}

// Layer is one layer in a drawing.
type Layer struct {
	Strokes []Stroke
	// index?
}

// Stroke is a single continous brush stroke.
type Stroke struct {
	// BrushType is one of the predefined pencil types, e.g. "Ballpoint" or "PaintBrush"
	BrushType BrushType
	// BrushColor is one of the three available colors.
	BrushColor BrushColor
	// Padding - we do not know what this means and it seems to be "0" all the time.
	Padding uint32
	// BrushSize is the base size of the Brush (small, medium, large)
	BrushSize BrushSize
	// Unknown is ...well: unkown.
	Unknown float32
	// Dots are the coordionate points that make up this stroke.
	Dots []Dot
}

// Dot is a single point from a stroke.
type Dot struct {
	// X is the x-coordinate for this dot.
	X float32
	// Y is the -ycoordinate for this dot.
	Y float32
	// Speed is the speed with which the stylus moved across the screen.
	Speed float32
	// Tilt is the angle at which the stylus is positioned against
	// the screen. The angle is given in radians.
	Tilt float32
	// Width is the effective width of the brush.
	Width float32
	// Pressure is the amount of pressure applied to the stylus.
	// Value range is 0.0 trough 1.0
	Pressure float32
}

// Header starting a .rm binary file. This can help recognizing a .rm file.
const (
	headerV3  = "reMarkable .lines file, version=3          "
	headerV5  = "reMarkable .lines file, version=5          "
	headerLen = 43
)
