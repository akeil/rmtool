package fs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	rm "github.com/akeil/rmtool"
	"github.com/akeil/rmtool/internal/errors"
)

// Timestamp is the datatype for a UNIX timestamp in string format.
type Timestamp struct {
	time.Time
}

// Metadata holds the metadata for a notebook.
//
// This maps to the .metadata file from the tablet's file system.
type Metadata struct {
	// LastModified is the UTC date of the last edit as a Unix timestamp.
	LastModified Timestamp `json:"lastModified"`
	// Version is incremented with each change to the file, starting at "1".
	Version uint `json:"version"`
	// LastOpenedPage is set by the tablet to the page that was last viewed.
	LastOpenedPage uint `json:"lastOpenedPage"`
	// Parent is the ID of the parent folder.
	// It is empty if the notebook is located in the root folder.
	// It can also be set to the special value "trash" if the notebook is deleted.
	Parent string `json:"parent"`
	// Pinned is the bookmark/start for a notebook.
	Pinned bool `json:"pinned"`
	// Type tells whether this is a document or a folder.
	Type rm.NotebookType `json:"type"`
	// VisibleName is the display name for this item.
	VisibleName string `json:"visibleName"`
	// Deleted seems to be used internally by the tablet(?).
	Deleted bool `json:"deleted"`
	// MetadataModified seems to be used internally by the tablet(?).
	MetadataModified bool `json:"metadatamodified"`
	// Modified seems to be used internally by the tablet(?).
	Modified bool `json:"modified"`
	// Synced seems to be used internally by the tablet(?).
	Synced bool `json:"synced"`
}

func (m *Metadata) Validate() error {
	switch m.Type {
	case rm.DocumentType, rm.CollectionType:
		// ok
	default:
		return errors.NewValidationError("invalid type %v", m.Type)
	}

	if m.VisibleName == "" {
		return errors.NewValidationError("visible name must not be emtpty")
	}

	return nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	// Expects a string lke this: "1607462787637",
	// with the last four digits containing nanoseconds.
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	secs := int64(n / 1000)
	nanos := (int64(n) - (secs * 1000)) * 1000000
	ts := Timestamp{time.Unix(secs, nanos).UTC()}

	*t = ts
	return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	nanos := t.UnixNano()
	millis := nanos / 1000000

	s := fmt.Sprintf("%d", millis)
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}
