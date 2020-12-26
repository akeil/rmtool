package rm

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestReadContent(t *testing.T) {
	path := "./testdata/25e3a0ce-080a-4389-be2a-f6aa45ce0207.content"
	var c Content
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}

	err = json.Unmarshal(data, &c)
	if err != nil {
		t.Error(err)
	}

	if c.FileType != Notebook {
		t.Errorf("unexpected file type")
	}

	expectedPageCount := 8
	if c.PageCount != expectedPageCount {
		t.Errorf("unexpected page count: %v != %v", c.PageCount, expectedPageCount)
	}

	if len(c.Pages) != expectedPageCount {
		t.Errorf("unexpected number of page ids")
	}
}

// TestValidateContent asserts that a Content struct initialized with NewConent
// meets the minimum requirements for validation.
func TestValidateContent(t *testing.T) {
	c := NewContent(Notebook)
	err := c.Validate()
	if err != nil {
		t.Error(err)
	}

	c.FileType = FileType(100) // does not exist
	if c.Validate() == nil {
		t.Errorf("Invalid FileType not detected")
	}

	c.FileType = Pdf
	c.Orientation = Orientation(100)
	if c.Validate() == nil {
		t.Errorf("Invalid Orientation not detected")
	}
	c.Orientation = Landscape

	c.PageCount = 100
	if c.Validate() == nil {
		t.Errorf("Mismatching number of pages not detected")
	}
	c.PageCount = 0

	c.Pages = append(c.Pages, "a-page-id")
	if c.Validate() == nil {
		t.Errorf("Mismatching number of pages not detected")
	}

}

func TestReadPageMetadata(t *testing.T) {
	path := "./testdata/25e3a0ce-080a-4389-be2a-f6aa45ce0207/0408f802-a07c-45c7-8382-7f8a36645fda-metadata.json"
	var p PageMetadata
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		t.Error(err)
	}

	if len(p.Layers) != 1 {
		t.Errorf("unexpected number of layers")
	}

	if p.Layers[0].Name != "Layer 1" {
		t.Errorf("unexpected layer name")
	}
}
