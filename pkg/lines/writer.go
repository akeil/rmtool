package lines

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// MarshalBinary returns the byte representation of the drawing.
func (d *Drawing) MarshalBinary() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := write(io.Writer(buf), d)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WriteDrawing writes the given drawing to the given writer.
func WriteDrawing(w io.Writer, d *Drawing) error {
	return write(w, d)
}

// Write writes the given drawing to the given writer
func write(w io.Writer, d *Drawing) error {
	err := writeHeader(w, d)
	if err != nil {
		return err
	}

	// the number of layers:
	numLayers := uint32(d.NumLayers())
	err = binary.Write(w, endianess, numLayers)
	if err != nil {
		return err
	}

	for _, l := range d.Layers {
		err = writeLayer(w, l)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeHeader(w io.Writer, d *Drawing) error {
	var h string
	switch d.Version {
	case V3:
		h = headerV3
	case V5:
		h = headerV5
	default:
		return fmt.Errorf("invalid version %v", d.Version)
	}

	_, err := w.Write([]byte(h))
	return err
}

func writeLayer(w io.Writer, l Layer) error {
	numStrokes := uint32(len(l.Strokes))
	err := binary.Write(w, endianess, numStrokes)
	if err != nil {
		return err
	}

	for _, s := range l.Strokes {
		err = writeStroke(w, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeStroke(w io.Writer, s Stroke) error {
	err := binary.Write(w, endianess, s.BrushType)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, s.BrushColor)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, s.Padding)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, s.BrushSize)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, s.Unknown)
	if err != nil {
		return err
	}

	numDots := uint32(len(s.Dots))
	err = binary.Write(w, endianess, numDots)
	if err != nil {
		return err
	}

	for _, d := range s.Dots {
		err = writeDot(w, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeDot(w io.Writer, d Dot) error {
	err := binary.Write(w, endianess, d.X)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, d.Y)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, d.Speed)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, d.Tilt)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, d.Width)
	if err != nil {
		return err
	}

	err = binary.Write(w, endianess, d.Pressure)
	if err != nil {
		return err
	}

	return nil
}
