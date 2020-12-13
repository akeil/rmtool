package rmtool

import (
	"io/ioutil"
	"testing"
)

func TestRead(t *testing.T) {
	path := "./testdata/faf24233-a397-409e-8993-914113af7d54/3ef76edb-f118-47f0-8e0c-d79ac63df4d6.rm"
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read rm file %q. Error: %v", path, err)
	}

	p, err := ReadDrawing(data)
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
