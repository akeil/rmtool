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

// NotebookType is used to distinguish betweeen documents and folders.
type NotebookType int

const (
	DocumentType NotebookType = iota
	CollectionType
)

// Orientation is the layout of a notebook page.
// It can be Portrait or Landscape.
type Orientation int

const (
	Portrait Orientation = iota
	Landscape
)

// FileType are the different types of supported content for a notebook.
type FileType int

const (
	Notebook FileType = iota
	Epub
	Pdf
)

// Metadata holds the metadata for a notebook.
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
	Type NotebookType `json:"type"`
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

// Content holds the data from the remarkable `.content` file.
// It describes the content for a notebook, specifically the sequence of pages.
// Collections have an empty content object.
type Content struct {
	// FileType is the type of content (i.e. handwritten Notebook or PDF, EPUB).
	FileType FileType `json:"fileType"`
	// Orientation gives the base layout orientation.
	// Individual pages can have a different orientation.
	Orientation Orientation `json:"orientation"`
	// PageCount is the number of pages in this notebooks.
	PageCount uint `json:"pageCount"`
	// Pages is a list of page IDs in the correct order.
	Pages []string `json:"pages"`
	// CoverPageNumber is the page that should be used as the cover in the UI.
	CoverPageNumber int `json:"coverPageNumber"`
}

// PageMetadata holds the layer information for a single page.
type PageMetadata struct {
	// Layers is the list of layers for a page.
	Layers []LayerMetadata `json:"layers"`
}

// LayerMetadata describes one layer.
type LayerMetadata struct {
	// Name is the display name for this layer.
	Name string `json:"name"`
	// TODO: visible y/n?
}

// ReadMetadata reads a Metadata struct from the given JSON file.
//
// Note that you can also use `json.Unmarshal(data, m)`.
// TODO - remove this?
func ReadMetadata(path string) (Metadata, error) {
	var m Metadata
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return m, err
	}

	err = json.Unmarshal(data, &m)
	return m, err
}

// TODO - remove this?
func ReadContent(path string) (Content, error) {
	var c Content
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(data, &c)
	return c, err
}

// TODO - remove this?
func ReadPageMetadata(path string) (PageMetadata, error) {
	var p PageMetadata
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return p, err
	}

	err = json.Unmarshal(data, &p)
	return p, err
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

func (o *Orientation) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var x Orientation
	switch s {
	case "portrait":
		x = Portrait
	case "landscape":
		x = Landscape
	default:
		return fmt.Errorf("invalid notebook type %q", s)
	}

	*o = x
	return nil
}

func (o Orientation) MarshalJSON() ([]byte, error) {
	s := o.String()

	if s == "UNKNOWN" {
		return nil, fmt.Errorf("invalid notebook type %v", o)
	}

	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}

func (o Orientation) String() string {
	switch o {
	case Portrait:
		return "portrait"
	case Landscape:
		return "landscape"
	default:
		return "UNKNOWN"
	}
}

func (f *FileType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var ft FileType
	switch s {
	case "notebook":
		ft = Notebook
	case "epub":
		ft = Epub
	case "pdf":
		ft = Pdf
	default:
		return fmt.Errorf("invalid file type %q", s)
	}

	*f = ft
	return nil
}

func (f FileType) MarshalJSON() ([]byte, error) {
	s := f.String()
	if s == "UNKNOWN" {
		return nil, fmt.Errorf("invalid file type %v", f)
	}

	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}

func (f FileType) String() string {
	switch f {
	case Notebook:
		return "notebook"
	case Epub:
		return "epub"
	case Pdf:
		return "pdf"
	default:
		return "UNKNOWN"
	}
}
