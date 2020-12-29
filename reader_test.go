package rm

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestRead(t *testing.T) {
	path := "./testdata/25e3a0ce-080a-4389-be2a-f6aa45ce0207/0408f802-a07c-45c7-8382-7f8a36645fda.rm"
	r, err := os.Open(path)
	if err != nil {
		t.Errorf("cannot read rm file %q. Error: %v", path, err)
	}
	defer r.Close()

	p, err := ReadDrawing(r)
	if err != nil {
		t.Error(err)
	}

	if p.Version != V5 {
		t.Errorf("wrong version number")
	}

	actualLayers := p.NumLayers()
	expectedLayers := 1
	if actualLayers != expectedLayers {
		t.Errorf("wrong layer count (%v != %v)", actualLayers, expectedLayers)
	}
}

func TestWriteRead(t *testing.T) {
	d := &Drawing{
		Version: V5,
		Layers: []Layer{
			Layer{
				Strokes: []Stroke{
					Stroke{
						BrushSize:  Medium,
						BrushType:  PencilV5,
						BrushColor: Gray,
						Dots: []Dot{
							Dot{
								Pressure: 1.0,
								Speed:    0.4,
								Tilt:     0.2,
								Width:    4.7,
								X:        100.0,
								Y:        200.0,
							},
							Dot{
								Pressure: 1.0,
								Speed:    0.5,
								Tilt:     0.3,
								Width:    2.7,
								X:        110.0,
								Y:        210.0,
							},
						},
					},
				},
			},
			Layer{
				Strokes: []Stroke{
					Stroke{
						BrushSize:  Large,
						BrushType:  BallpointV5,
						BrushColor: Black,
						Dots: []Dot{
							Dot{
								Pressure: 0.85,
								Speed:    0.74,
								Tilt:     0.65,
								Width:    5.7,
								X:        500.0,
								Y:        400.0,
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := WriteDrawing(io.Writer(&buf), d)
	if err != nil {
		t.Error(err)
	}

	x, err := ReadDrawing(io.Reader(&buf))
	if err != nil {
		t.Error(err)
	}

	if x.Version != d.Version {
		t.Errorf("version mismatch afer r/w cycle")
	}

	if len(x.Layers) != len(d.Layers) {
		t.Errorf("layer mismatch afer r/w cycle")
	}

	if x.Layers[0].Strokes[0].BrushType != d.Layers[0].Strokes[0].BrushType {
		t.Errorf("brush type mismatch afer r/w cycle")
	}

	if x.Layers[0].Strokes[0].BrushSize != d.Layers[0].Strokes[0].BrushSize {
		t.Errorf("brush size mismatch afer r/w cycle")
	}

	if x.Layers[0].Strokes[0].Dots[0].X != d.Layers[0].Strokes[0].Dots[0].X {
		t.Errorf("dot mismatch afer r/w cycle")
	}
	if x.Layers[0].Strokes[0].Dots[0].Y != d.Layers[0].Strokes[0].Dots[0].Y {
		t.Errorf("dot mismatch afer r/w cycle")
	}
	if x.Layers[0].Strokes[0].Dots[0].Pressure != d.Layers[0].Strokes[0].Dots[0].Pressure {
		t.Errorf("dot mismatch afer r/w cycle")
	}
}
