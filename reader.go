package rmtool

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var endianess = binary.LittleEndian

// UnmarshalBinary reads a reMarkable drawing from the given bytes.
func (d *Drawing) UnmarshalBinary(data []byte) error {
	err := readInto(data, d)
	if err != nil {
		return err
	}

	return nil
}

// Read creates a new reMarkable drawing from the given bytes.
func Read(data []byte) (*Drawing, error) {
	d := newDrawing()
	err := d.UnmarshalBinary(data)
	return d, err
}

// readInto reads the given byte data into the given drawing.
func readInto(data []byte, d *Drawing) error {
	r := newReader(data)

	err := r.readHeader()
	if err != nil {
		return err
	}
	d.Version = r.version

	nLayers, err := r.readNumber()
	if err != nil {
		return err
	}

	d.Layers = make([]Layer, nLayers)
	for i := uint32(0); i < nLayers; i++ {
		nStrokes, err := r.readNumber()
		d.Layers[i].Strokes = make([]Stroke, nStrokes)
		if err != nil {
			return err
		}

		for j := uint32(0); j < nStrokes; j++ {
			s, err := r.readStroke()
			if err != nil {
				return err
			}
			d.Layers[i].Strokes[j] = s
		}
	}

	return nil
}

type reader struct {
	bytes.Reader
	version Version
}

// newReader creates a new page reader.
func newReader(data []byte) reader {
	r := bytes.NewReader(data)

	// V5 will be replaced after reading the header
	return reader{*r, V5}
}

// readHeader and check if it is one of the supported headers.
func (r *reader) readHeader() error {
	buf := make([]byte, headerLen)

	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	if n != headerLen {
		return fmt.Errorf("unexpected header size")
	}

	switch string(buf) {
	case headerV3:
		r.version = V3
	case headerV5:
		r.version = V5
	default:
		return fmt.Errorf("unsupported header")
	}

	return nil
}

// readLine reads a Stroke (incl. Dots) from the reader.
func (r *reader) readStroke() (Stroke, error) {
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

	// additional attribute in v5 only
	if r.version == V5 {
		err := binary.Read(r, endianess, &s.Unknown)
		if err != nil {
			return s, fmt.Errorf("failed to read line")
		}
	}

	err = binary.Read(r, endianess, &s.BrushSize)
	if err != nil {
		return s, fmt.Errorf("failed to read brush size")
	}

	nDots, err := r.readNumber()
	if err != nil {
		return s, fmt.Errorf("failed to read number of dots")
	}

	s.Dots = make([]Dot, nDots)
	for i := uint32(0); i < nDots; i++ {
		d, err := r.readDot()
		if err != nil {
			return s, err
		}
		s.Dots[i] = d
	}

	return s, nil
}

// readNumber reads a uint32 from the reader.
func (r *reader) readNumber() (uint32, error) {
	var n uint32
	err := binary.Read(r, endianess, &n)
	return n, err
}

// readDot reads a Dot struct from the reader.
func (r *reader) readDot() (Dot, error) {
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

	err = binary.Read(r, endianess, &d.Direction)
	if err != nil {
		return d, fmt.Errorf("failed to read direction")
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
