package rm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

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

type TemplateSize int

const (
	TemplateNoSize TemplateSize = iota
	TemplateSmall
	TemplateMedium
	TemplateLarge
)

const maxLayers = 5

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
	PageCount int `json:"pageCount"`
	// Pages is a list of page IDs in the correct order.
	Pages []string `json:"pages"`
	// CoverPageNumber is the page that should be used as the cover in the UI.
	CoverPageNumber int `json:"coverPageNumber"`
}

func NewContent(f FileType) *Content {
	return &Content{
		FileType:    f,
		Orientation: Portrait,
		PageCount:   0,
		Pages:       make([]string, 0),
	}
}

func (c *Content) Validate() error {
	switch c.FileType {
	case Notebook, Pdf, Epub:
		// ok
	default:
		return NewValidationError("invalid file type %v", c.FileType)
	}

	switch c.Orientation {
	case Portrait, Landscape: // ok
	default:
		return NewValidationError("invalid orientation %v", c.Orientation)
	}

	if c.PageCount != len(c.Pages) {
		return NewValidationError("pageCount does not match number of pages %v != %v", c.PageCount, len(c.Pages))
	}

	return nil
}

// PageMetadata holds the layer information for a single page.
type PageMetadata struct {
	// Layers is the list of layers for a page.
	Layers []LayerMetadata `json:"layers"`
}

func (p PageMetadata) Validate() error {
	if p.Layers == nil {
		return NewValidationError("no layers defined")
	}
	if len(p.Layers) == 0 {
		return NewValidationError("no layers defined")
	}
	if len(p.Layers) > maxLayers {
		return NewValidationError("maximum number of layers exceeded")
	}

	for _, l := range p.Layers {
		err := l.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// LayerMetadata describes one layer.
type LayerMetadata struct {
	// Name is the display name for this layer.
	Name string `json:"name"`
	// TODO: visible y/n?
}

func (l LayerMetadata) Validate() error {
	if l.Name == "" {
		return NewValidationError("layer name must not be empty")
	}

	return nil
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

type Pagedata struct {
	Orientation Orientation
	Template    string
	Size        TemplateSize
	Text        string
}

// HasTemplate tells if the page has a (visible) background template.
func (p *Pagedata) HasTemplate() bool {
	return p.Text != "Blank" && p.Text != ""
}

func (p *Pagedata) Validate() error {
	// TODO implement
	return nil
}

func ReadPagedata(r io.Reader) ([]Pagedata, error) {
	pd := make([]Pagedata, 0)
	s := bufio.NewScanner(r)

	var text string
	var err error
	var size TemplateSize
	var layout Orientation
	var parts []string
	for s.Scan() {
		text = s.Text()
		err = s.Err()
		if err != nil {
			return pd, err
		}
		// TODO: assumes that empty lines are allowed - correct?
		if text == "" {
			continue
		}

		// Special case: some templates do not have the orientation prefix
		switch text {
		case "Blank",
			"Isometric",
			"Perspective1",
			"Perspective2":
			pd = append(pd, Pagedata{
				Orientation: Portrait,
				Template:    text,
				Size:        TemplateMedium,
				Text:        text,
			})
		default:
			// TODO some templates have no size
			parts = strings.SplitN(text, " ", 3)
			if len(parts) != 3 {
				return pd, fmt.Errorf("invalid pagedata line: %q", text)
			}
			size = size.FromString(parts[2])
			layout = layout.fromString(parts[0])
			pd = append(pd, Pagedata{
				Orientation: layout,
				Template:    parts[1],
				Size:        size,
				Text:        text,
			})
		}
	}

	return pd, nil
}

func (t TemplateSize) FromString(s string) TemplateSize {
	switch s {
	case "S", "small":
		return TemplateSmall
	case "M", "medium", "med":
		return TemplateMedium
	case "L", "large":
		return TemplateLarge
	}
	return TemplateNoSize
}

func (o Orientation) fromString(s string) Orientation {
	switch s {
	case "P":
		return Portrait
	case "LS":
		return Landscape
	default:
		return Portrait
	}
}

func (o Orientation) toString() string {
	switch o {
	case Portrait:
		return "P"
	case Landscape:
		return "LS"
	default:
		return ""
	}
}
