package rm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var endianess = binary.LittleEndian

// UnmarshalBinary reads a reMarkable drawing from the given bytes.
func (d *Drawing) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	err := read(r, d)
	if err != nil {
		return err
	}

	return nil
}

// ReadDrawing creates a new reMarkable drawing from the given reader.
func ReadDrawing(r io.Reader) (*Drawing, error) {
	d := &Drawing{}
	err := read(r, d)
	return d, err
}

// read reads the given byte data into the given drawing.
func read(r io.Reader, d *Drawing) error {
	version, err := readHeader(r)
	if err != nil {
		return err
	}
	d.Version = version

	nLayers, err := readNumber(r)
	if err != nil {
		return err
	}

	d.Layers = make([]Layer, nLayers)
	for i := uint32(0); i < nLayers; i++ {
		nStrokes, err := readNumber(r)
		d.Layers[i].Strokes = make([]Stroke, nStrokes)
		if err != nil {
			return err
		}

		for j := uint32(0); j < nStrokes; j++ {
			s, err := readStroke(r, version)
			if err != nil {
				return err
			}
			d.Layers[i].Strokes[j] = s
		}
	}

	return nil
}

// readHeader and check if it is one of the supported headers.
func readHeader(r io.Reader) (Version, error) {
	var v Version
	buf := make([]byte, headerLen)

	n, err := r.Read(buf)
	if err != nil {
		return v, err
	}
	if n != headerLen {
		return v, fmt.Errorf("unexpected header size")
	}

	switch string(buf) {
	case headerV3:
		v = V3
	case headerV5:
		v = V5
	default:
		return v, fmt.Errorf("unsupported header")
	}

	return v, nil
}

// readStroke reads a Stroke (incl. Dots) from the reader.
func readStroke(r io.Reader, v Version) (Stroke, error) {
	var s Stroke

	err := binary.Read(r, endianess, &s.BrushType)
	if err != nil {
		return s, fmt.Errorf("failed to read brush type")
	}

	err = binary.Read(r, endianess, &s.BrushColor)
	if err != nil {
		return s, fmt.Errorf("failed to read brush color")
	}

	err = binary.Read(r, endianess, &s.Padding)
	if err != nil {
		return s, fmt.Errorf("failed to read padding")
	}

	err = binary.Read(r, endianess, &s.BrushSize)
	if err != nil {
		return s, fmt.Errorf("failed to read brush size")
	}

	// additional attribute in v5 only
	if v == V5 {
		err := binary.Read(r, endianess, &s.Unknown)
		if err != nil {
			return s, fmt.Errorf("failed to read line")
		}
	}

	nDots, err := readNumber(r)
	if err != nil {
		return s, fmt.Errorf("failed to read number of dots")
	}

	s.Dots = make([]Dot, nDots)
	for i := uint32(0); i < nDots; i++ {
		d, err := readDot(r)
		if err != nil {
			return s, err
		}
		s.Dots[i] = d
	}

	return s, nil
}

// readNumber reads a uint32 from the reader.
func readNumber(r io.Reader) (uint32, error) {
	var n uint32
	err := binary.Read(r, endianess, &n)
	return n, err
}

// readDot reads a Dot struct from the reader.
func readDot(r io.Reader) (Dot, error) {
	var d Dot

	err := binary.Read(r, endianess, &d.X)
	if err != nil {
		return d, fmt.Errorf("failed to read X-coordinate")
	}

	err = binary.Read(r, endianess, &d.Y)
	if err != nil {
		return d, fmt.Errorf("failed to read Y-coordinate")
	}

	err = binary.Read(r, endianess, &d.Speed)
	if err != nil {
		return d, fmt.Errorf("failed to read speed")
	}

	err = binary.Read(r, endianess, &d.Tilt)
	if err != nil {
		return d, fmt.Errorf("failed to read tilt")
	}

	err = binary.Read(r, endianess, &d.Width)
	if err != nil {
		return d, fmt.Errorf("failed to read width")
	}

	err = binary.Read(r, endianess, &d.Pressure)
	if err != nil {
		return d, fmt.Errorf("failed to read pressure")
	}

	return d, nil
}
