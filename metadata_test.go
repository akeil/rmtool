package rmtool

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReadMetadata(t *testing.T) {
	path := "./testdata/faf24233-a397-409e-8993-914113af7d54.metadata"
	m, err := ReadMetadata(path)

	if err != nil {
		t.Error(err)
	}

	if m.Deleted {
		t.Errorf("unexpected value for deleted")
	}
	// TODO: timestamp

	if m.Version != 6 {
		t.Errorf("unexpected value for version")
	}

	if m.VisibleName != "Notebook" {
		t.Errorf("unexpected value for version")
	}

	// 2020-12-08 22:26:27.637
	if m.LastModified.Year() != 2020 {
		t.Errorf("unexpected value for lastModified")
	}
	if m.LastModified.Second() != 27 {
		t.Errorf("unexpected value for lastModified")
	}
	if m.LastModified.Nanosecond() != 637_000_000 {
		t.Errorf("unexpected value for lastModified")
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

func TestReadPageMetadata(t *testing.T) {
	path := "./testdata/faf24233-a397-409e-8993-914113af7d54/3ef76edb-f118-47f0-8e0c-d79ac63df4d6-metadata.json"
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
