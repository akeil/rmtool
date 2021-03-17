package fs

import (
	"encoding/json"
	"testing"
	"time"

	rm "github.com/akeil/rmtool"
)

func TestReadMetadata(t *testing.T) {
	jsonStr := `{
        "deleted": false,
        "lastModified": "1608230074814",
        "lastOpenedPage": 5,
        "metadatamodified": false,
        "modified": false,
        "parent": "033cab93-8da0-4672-b63b-31d3252a8dc9",
        "pinned": false,
        "synced": true,
        "type": "DocumentType",
        "version": 42,
        "visibleName": "Test"
    }`

	var m Metadata
	err := json.Unmarshal([]byte(jsonStr), &m)

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
	if m.LastModified.Nanosecond() != 814000000 {
		t.Errorf("unexpected value for lastModified (Nanosecond): %v", m.LastModified.Nanosecond())
	}

	if m.Type != rm.DocumentType {
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
		Type:             rm.DocumentType,
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

func TestValidateMetadata(t *testing.T) {
	m := &Metadata{
		Type:        rm.DocumentType,
		VisibleName: "abc",
	}

	err := m.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}

	m.Type = rm.NotebookType(100)
	err = m.Validate()
	if err == nil {
		t.Errorf("Invalid type not detected")
	}
	m.Type = rm.CollectionType

	m.VisibleName = ""
	err = m.Validate()
	if err == nil {
		t.Errorf("Invalid VisibleName not detected")
	}
	m.VisibleName = "abc"
}
