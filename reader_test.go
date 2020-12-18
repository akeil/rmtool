package rm

import (
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
