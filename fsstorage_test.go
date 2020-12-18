package rm

import (
	"testing"
)

func TestReadNotebook(t *testing.T) {
	s := NewFilesystemStorage("testdata")
	id := "25e3a0ce-080a-4389-be2a-f6aa45ce0207"

	n, err := ReadNotebook(s, id)
	if err != nil {
		t.Error(err)
	}

	if n.Meta.VisibleName != "Test" {
		t.Errorf("unexpected notebook name")
	}

	for _, p := range n.Pages {
		_, err = s.ReadDrawing(n.ID, p.ID)
		if err != nil {
			t.Error(err)
		}
	}
}
