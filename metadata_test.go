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

    t.Errorf("%v", m)
}

func TestMarshalMetadata(t *testing.T) {
    d := time.Date(2020, 12, 13, 7, 23, 43, 589000000, time.UTC)
    m := Metadata{
        Deleted: true,
        LastModified: Timestamp{d},
        VisibleName: "Test Notebook",
    }

    data, err := json.Marshal(m)
    if err != nil {
		t.Error(err)
	}

    s := string(data)
    t.Errorf(s)
}
