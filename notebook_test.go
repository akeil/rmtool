package rm

import (
	"testing"
)

func TestReadNotebook(t *testing.T) {
	base := "./testdata"
	id := "25e3a0ce-080a-4389-be2a-f6aa45ce0207"

	n := NewNotebook(base, id)
	err := n.Read()
	if err != nil {
		t.Error(err)
	}

	if n.Meta.VisibleName != "Test" {
		t.Errorf("unexpected notebook name")
	}

	for _, p := range n.Pages {
		err = p.ReadDrawing()
		if err != nil {
			t.Error(err)
		}
	}
}
