package rm

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReadMetadata(t *testing.T) {
	path := "./testdata/25e3a0ce-080a-4389-be2a-f6aa45ce0207.metadata"
	m, err := ReadMetadata(path)

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
		Deleted:      true,
		LastModified: Timestamp{d},
		Type:         DocumentType,
		VisibleName:  "Test Notebook",
	}

	_, err := json.Marshal(m)
	if err != nil {
		t.Error(err)
	}
}

func TestReadContent(t *testing.T) {
	path := "./testdata/25e3a0ce-080a-4389-be2a-f6aa45ce0207.content"
	c, err := ReadContent(path)

	if err != nil {
		t.Error(err)
	}

	if c.FileType != Notebook {
		t.Errorf("unexpected file type")
	}

	expectedPageCount := uint(8)
	if c.PageCount != expectedPageCount {
		t.Errorf("unexpected page count: %v != %v", c.PageCount, expectedPageCount)
	}

	if uint(len(c.Pages)) != expectedPageCount {
		t.Errorf("unexpected number of page ids")
	}
}

func TestReadPageMetadata(t *testing.T) {
	path := "./testdata/25e3a0ce-080a-4389-be2a-f6aa45ce0207/0408f802-a07c-45c7-8382-7f8a36645fda-metadata.json"
	p, err := ReadPageMetadata(path)

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
