package lines

import (
	"fmt"
	"math"

	"akeil.net/akeil/rm/internal/errors"
)

// Validate checks this drawing and all layers, strokes and dots for valid data.
// Returns an error if invalid data is found, nil if everything is fine.
func (d *Drawing) Validate() error {
	if d.Version != V3 && d.Version != V5 {
		return fmt.Errorf("invalid version: %v", d.Version)
	}

	if d.Layers == nil || len(d.Layers) == 0 {
		return errors.NewValidationError("drawing must have at least one layer")
	}

	for _, l := range d.Layers {
		err := l.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate checks a layer and all associated strokes and dots for valid data.
// Returns an error if invalid data is found, nil if everything is fine.
func (l *Layer) Validate() error {
	if l.Strokes == nil {
		return nil
	}

	for _, s := range l.Strokes {
		err := s.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate checks a stroke and the associated dots for valid data.
// Returns an error if invalid data is found, nil if everything is fine.
func (s *Stroke) Validate() error {
	err := validateBrushType(s.BrushType)
	if err != nil {
		return err
	}

	switch s.BrushColor {
	case Black, Gray, White:
		// valid
	default:
		return fmt.Errorf("invalid color: %v", s.BrushColor)
	}

	switch s.BrushSize {
	case Small, Medium, Large:
		// valid
	default:
		return fmt.Errorf("invalid brush size: %v", s.BrushSize)
	}

	if s.Dots == nil {
		return nil
	}

	for _, d := range s.Dots {
		err = d.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate checks a dot for valid data.
// Returns an error if invalid data is found, nil if everything is fine.
func (d *Dot) Validate() error {
	if d.X < 0 || d.X > MaxWidth {
		return fmt.Errorf("invalid x-coordinate: %v", d.X)
	}

	if d.Y < 0 || d.Y > MaxHeight {
		return fmt.Errorf("invalid y-coordinate: %v", d.Y)
	}

	// TODO: not sure what the MAX value for Speed should be
	if d.Speed < 0 {
		return fmt.Errorf("invalid speed value: %v", d.Speed)
	}

	// TODO: Encountered tilt values outside of the intervals
	// So thie below validations rules seem to be wrong?

	// 0..90 degrees
	max0 := rad(90)
	interval0 := d.Tilt >= 0 && d.Tilt <= max0
	// 270..360 degrees
	min1 := rad(270)
	max1 := rad(360)
	interval1 := d.Tilt >= min1 && d.Tilt <= max1
	if !(interval0 || interval1) {
		return fmt.Errorf("invalid tilt value: %v", d.Tilt)
	}

	// TODO: not sure what the MAX value for width should be
	if d.Width < 0 {
		return fmt.Errorf("invalid width value: %v", d.Width)
	}

	if d.Pressure < 0 || d.Pressure > 1 {
		return fmt.Errorf("invalid pressure value: %v", d.Pressure)
	}

	return nil
}

func rad(deg float32) float32 {
	return deg * (math.Pi / 180)
}

func validateBrushType(b BrushType) error {
	switch b {
	case PaintBrush,
		Pencil,
		Ballpoint,
		Marker,
		Fineliner,
		Highlighter,
		Eraser,
		MechanicalPencil,
		EraseArea,
		PaintBrushV5,
		MechanicalPencilV5,
		PencilV5,
		BallpointV5,
		MarkerV5,
		FinelinerV5,
		HighlighterV5:
		return nil
	default:
		return fmt.Errorf("invalid brush type: %v", b)
	}
}
