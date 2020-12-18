package rm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"
)

// Timestampp is the datatype for a UNIX timestamp in string format.
type Timestamp struct {
	time.Time
}

type NotebookType int

const (
	DocumentType = iota
	CollectionType
)

// Metadata holds the metadata for a notebook.
type Metadata struct {
	Deleted          bool         `json:"deleted"`
	LastModified     Timestamp    `json:"lastModified"`
	LastOpenedPage   uint         `json:"lastOpenedPage"`
	Metadatamodified bool         `json:"metadatamodified"`
	Modified         bool         `json:"modified"`
	Parent           string       `json:"parent"`
	Pinned           bool         `json:"pinned"`
	Synced           bool         `json:"synced"`
	Type             NotebookType `json:"type"`
	Version          uint         `json:"version"`
	VisibleName      string       `json:"visibleName"`
}

// ReadMetadata reads a Metadata struct from the given JSON file.
//
// Note that you can also use `json.Unmarshal(data, m)`.
func ReadMetadata(path string) (Metadata, error) {
	var m Metadata
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return m, err
	}

	err = json.Unmarshal(data, &m)
	return m, err
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	// expects a string lke this: 1607462787637
	// with the last for digits containing nanoseconds.
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	secs := int64(n / 1_000)
	nanos := (int64(n) - (secs * 1_000)) * 1_000_000
	ts := Timestamp{time.Unix(secs, nanos)}

	*t = ts
	return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	nanos := t.UnixNano()
	millis := nanos / 1_000_000

	s := fmt.Sprintf("%d", millis)
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}

func (n *NotebookType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var nt NotebookType
	switch s {
	case "DocumentType":
		nt = DocumentType
	case "CollectionType":
		nt = CollectionType
	default:
		return fmt.Errorf("invalid notebook type %q", s)
	}

	*n = nt
	return nil
}

func (n NotebookType) MarshalJSON() ([]byte, error) {
	var s string
	switch n {
	case DocumentType:
		s = "DocumentType"
	case CollectionType:
		s = "CollectionType"
	default:
		return nil, fmt.Errorf("invalid notebook type %v", n)
	}

	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}

type Content struct {
	FileType  string   `json:"filetype"`
	PageCount int      `json:"pageCount"`
	Pages     []string `json:"pages"`
}

func ReadContent(path string) (Content, error) {
	var c Content
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(data, &c)
	return c, err
}

type PageMetadata struct {
	Layers []LayerMetadata `json:"layers"`
}

type LayerMetadata struct {
	Name string `json:"name"`
}

func ReadPageMetadata(path string) (PageMetadata, error) {
	var p PageMetadata
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return p, err
	}

	err = json.Unmarshal(data, &p)
	return p, err
}
