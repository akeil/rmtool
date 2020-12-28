package rm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
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

type LineHeight int

const (
	LineHeightDefault LineHeight = -1
	LineHeightSmall   LineHeight = 100
	LineHeightMedium  LineHeight = 150
	LineHeightLarge   LineHeight = 200
)

type TextAlign int

const (
	AlignLeft TextAlign = iota
	AlignJustify
)

const maxLayers = 5
const defaultCoverPage = -1

// Content holds the data from the remarkable `.content` file.
// It describes the content for a notebook, specifically the sequence of pages.
// Collections have an empty content object.
type Content struct {
	DummyDocument bool `json:"dummyDocument"`

	ExtraMetadata ExtraMetadata `json:"extraMetadata"`
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

	// not sure if these are relevant

	// FontName for EPUB, empty to use default (probably a list w/ supported font names)
	FontName string `json:"fontName"`
	// LineHeight always seems to be -1 / 150 / 200 / 100?
	LineHeight LineHeight `json:"lineHeight"`
	// MArgins are the page margins (left/right?) for EPUB and PDF files, default is 100 (180 for PDF?)
	Margins int `json:"margins"`
	// TextAlignment for EPUB, left or justify
	TextAlignment TextAlign `json:"textAlignment"`
	// TextScale for EPUB, default is 1.0,
	TextScale float32   `json:"textScale"`
	Transform Transform `json:"transform"`
}

func NewContent(f FileType) *Content {
	return &Content{
		DummyDocument:   false,
		ExtraMetadata:   NewExtraMetadata(),
		CoverPageNumber: defaultCoverPage,
		FileType:        f,
		Orientation:     Portrait,
		PageCount:       0,
		Pages:           make([]string, 0),
		// default values taken from a sample file
		FontName:      "",
		LineHeight:    LineHeightDefault,
		Margins:       100,
		TextAlignment: AlignLeft,
		TextScale:     1.0,
		Transform:     NewTransform(),
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

	// Cover page may be -1 (=not set)
	// or an existing page
	if c.CoverPageNumber != defaultCoverPage {
		if c.CoverPageNumber < 1 || c.CoverPageNumber > c.PageCount {
			return NewValidationError("cover page %v is not an existing page", c.CoverPageNumber)
		}
	}

	// TODO validate font names
	// TODO validate LineHeight
	// TODO validate Margins
	// TODO: validate TextScale
	switch c.TextAlignment {
	case AlignLeft, AlignJustify:
		// ok
	default:
		return NewValidationError("invalid text align %v", c.TextAlignment)
	}

	return nil
}

type Transform struct {
	// TODO: these might also be floats
	// never seen anything other than identity transform with values set to 1 or 0
	M11 int `json:"m11"`
	M12 int `json:"m12"`
	M13 int `json:"m13"`
	M21 int `json:"m21"`
	M22 int `json:"m22"`
	M23 int `json:"m23"`
	M31 int `json:"m31"`
	M32 int `json:"m32"`
	M33 int `json:"m33"`
}

func NewTransform() Transform {
	return Transform{
		M11: 1,
		M22: 1,
		M33: 1,
	}
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

func (t *TextAlign) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var ta TextAlign
	switch s {
	case "left":
		ta = AlignLeft
	case "justify":
		ta = AlignJustify
	default:
		return fmt.Errorf("invalid text align %q", s)
	}

	*t = ta
	return nil
}

func (t TextAlign) MarshalJSON() ([]byte, error) {
	s := t.String()
	if s == "UNKNOWN" {
		return nil, fmt.Errorf("invalid text align type %v", t)
	}

	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}

func (t TextAlign) String() string {
	switch t {
	case AlignLeft:
		return "left"
	case AlignJustify:
		return "justify"
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

// TODO: parse from template string
func newPagedata() Pagedata {
	return Pagedata{
		Text: "Blank",
	}
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

func WritePagedata(pd []Pagedata, w io.Writer) error {
	for _, p := range pd {
		_, err := w.Write([]byte(p.Text + "\n"))
		if err != nil {
			return err
		}
	}
	return nil
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

type ExtraMetadata struct {
	LastBallpointColor       string
	LastBallpointSize        intStr
	LastBallpointv2Color     string
	LastBallpointv2Size      intStr
	LastBrushColor           string
	LastBrushThicknessScale  intStr
	LastCalligraphyColor     string
	LastCalligraphySize      intStr
	LastClearPageColor       string
	LastClearPageSize        intStr
	LastColor                string
	LastEraseSectionColor    string
	LastEraseSectionSize     intStr
	LastEraserColor          string
	LastEraserSize           intStr
	LastEraserThicknessScale intStr
	LastEraserTool           string //"Eraser"
	LastFinelinerColor       string
	LastFinelinerSize        intStr
	LastFinelinerv2Color     string
	LastFinelinerv2Size      intStr
	LastHighlighterColor     string
	LastHighlighterSize      intStr
	LastHighlighterv2Color   string
	LastHighlighterv2Size    intStr
	LastMarkerColor          string
	LastMarkerSize           intStr
	LastMarkerv2Color        string
	LastMarkerv2Size         intStr
	LastPaintbrushColor      string
	LastPaintbrushSize       intStr
	LastPaintbrushv2Color    string
	LastPaintbrushv2Size     intStr
	LastPen                  string // Ballpointv2
	LastPenColor             string
	LastPenThicknessScale    intStr
	LastPencil               string // SharpPencil
	LastPencilColor          string
	LastPencilSize           intStr
	LastPencilThicknessScale intStr
	LastPencilv2Color        string
	LastPencilv2Size         intStr
	LastReservedPenColor     string
	LastReservedPenSize      intStr
	LastSelectionToolColor   string
	LastSelectionToolSize    intStr
	LastSharpPencilColor     string
	LastSharpPencilSize      intStr
	LastSharpPencilv2Color   string
	LastSharpPencilv2Size    intStr
	LastSolidPenColor        string
	LastSolidPenSize         intStr
	LastTool                 string // Ballpoint
	LastUndefinedColor       string
	LastUndefinedSize        intStr
	LastZoomToolColor        string
	LastZoomToolSize         intStr
	ThicknessScale           intStr
}

func NewExtraMetadata() ExtraMetadata {
	// default values taken from a sample file
	return ExtraMetadata{
		LastBallpointColor:       "Black",
		LastBallpointSize:        2,
		LastBallpointv2Color:     "Black",
		LastBallpointv2Size:      2,
		LastBrushColor:           "Black",
		LastBrushThicknessScale:  2,
		LastCalligraphyColor:     "Black",
		LastCalligraphySize:      2,
		LastClearPageColor:       "Black",
		LastClearPageSize:        2,
		LastColor:                "Black",
		LastEraseSectionColor:    "Black",
		LastEraseSectionSize:     2,
		LastEraserColor:          "Black",
		LastEraserSize:           2,
		LastEraserThicknessScale: 2,
		LastEraserTool:           "Eraser",
		LastFinelinerColor:       "Black",
		LastFinelinerSize:        2,
		LastFinelinerv2Color:     "Black",
		LastFinelinerv2Size:      2,
		LastHighlighterColor:     "Black",
		LastHighlighterSize:      2,
		LastHighlighterv2Color:   "Black",
		LastHighlighterv2Size:    2,
		LastMarkerColor:          "Black",
		LastMarkerSize:           2,
		LastMarkerv2Color:        "Black",
		LastMarkerv2Size:         2,
		LastPaintbrushColor:      "Black",
		LastPaintbrushSize:       2,
		LastPaintbrushv2Color:    "Black",
		LastPaintbrushv2Size:     2,
		LastPen:                  "Ballpointv2",
		LastPenColor:             "Black",
		LastPenThicknessScale:    2,
		LastPencil:               "SharpPencil",
		LastPencilColor:          "Black",
		LastPencilSize:           2,
		LastPencilThicknessScale: 2,
		LastPencilv2Color:        "Black",
		LastPencilv2Size:         2,
		LastReservedPenColor:     "Black",
		LastReservedPenSize:      2,
		LastSelectionToolColor:   "Black",
		LastSelectionToolSize:    2,
		LastSharpPencilColor:     "Black",
		LastSharpPencilSize:      2,
		LastSharpPencilv2Color:   "Black",
		LastSharpPencilv2Size:    2,
		LastSolidPenColor:        "Black",
		LastSolidPenSize:         2,
		LastTool:                 "Ballpoint",
		LastUndefinedColor:       "Black",
		LastUndefinedSize:        1,
		LastZoomToolColor:        "Black",
		LastZoomToolSize:         2,
		ThicknessScale:           2,
	}
}

type intStr int

func (is *intStr) UnmarshalJSON(b []byte) error {
	// expects a string lke this: 1607462787637
	// with the last for digits containing nanoseconds.
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	v, err := strconv.Atoi(s)

	*is = intStr(v)
	return nil
}

func (is intStr) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(fmt.Sprintf("%v", is))
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}
