package rm

import (
	"os"
	"testing"
)

func TestRead(t *testing.T) {
	path := "./testdata/faf24233-a397-409e-8993-914113af7d54/3ef76edb-f118-47f0-8e0c-d79ac63df4d6.rm"
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
