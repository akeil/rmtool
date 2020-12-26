package rm

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"
)

func TestReadMetadata(t *testing.T) {
	path := "./testdata/25e3a0ce-080a-4389-be2a-f6aa45ce0207.metadata"
	var m Metadata
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}

	err = json.Unmarshal(data, &m)

	if err != nil {
		t.Error(err)
	}

	if m.Deleted {
		t.Errorf("unexpected value for deleted")
	}
	// TODO: timestamp

	expectedVersion := uint(42)
	if m.Version != expectedVersion {
		t.Errorf("unexpected value for version: %v != %v", m.Version, expectedVersion)
	}

	expectedName := "Test"
	if m.VisibleName != expectedName {
		t.Errorf("unexpected value for visible name: %q != %q", m.VisibleName, expectedName)
	}

	// 2020-12-08 22:26:27.637
	if m.LastModified.Year() != 2020 {
		t.Errorf("unexpected value for lastModified (Year): %v", m.LastModified.Year())
	}
	if m.LastModified.Second() != 34 {
		t.Errorf("unexpected value for lastModified (Second): %v", m.LastModified.Second())
	}
	if m.LastModified.Nanosecond() != 814_000_000 {
		t.Errorf("unexpected value for lastModified (Nanosecond): %v", m.LastModified.Nanosecond())
	}

	if m.Type != DocumentType {
		t.Errorf("unexpected value for type")
	}
}

func TestMarshalMetadata(t *testing.T) {
	d := time.Date(2020, 12, 13, 7, 23, 43, 589000000, time.UTC)
	m := Metadata{
		LastModified:     Timestamp{d},
		Version:          4,
		LastOpenedPage:   0,
		Parent:           "parentID",
		Pinned:           true,
		Type:             DocumentType,
		VisibleName:      "Test Notebook",
		Deleted:          true,
		MetadataModified: false,
		Modified:         false,
		Synced:           false,
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Error(err)
	}

	var re Metadata
	err = json.Unmarshal(data, &re)
	if err != nil {
		t.Error(err)
	}

	if re.LastModified != m.LastModified {
		t.Errorf("Last modified changed in serialization: %v != %v", re.LastModified, m.LastModified)
		t.Fail()
	}
	if re.Version != m.Version {
		t.Fail()
	}
	if re.LastOpenedPage != m.LastOpenedPage {
		t.Fail()
	}
	if re.Parent != m.Parent {
		t.Fail()
	}
	if re.Pinned != m.Pinned {
		t.Fail()
	}
	if re.Type != m.Type {
		t.Fail()
	}
	if re.VisibleName != m.VisibleName {
		t.Fail()
	}
	if re.Deleted != m.Deleted {
		t.Fail()
	}
}

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
