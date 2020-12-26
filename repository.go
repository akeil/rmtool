package rm

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"akeil.net/akeil/rm/internal/logging"
)

// Repository is the interface for a storage backend.
//
// It can either represent local files copied from the tablet
// or notes accessed via the Cloud API.
//
// The repository offers methods to work on the metadata of items,
// allowing operations like rename or bookmark.
type Repository interface {
	// List returns a flat list of all entries in the repository.
	// The list is in no particular order - use BuildTree() to recreate the
	// tree structure with folders and subfolders.
	List() ([]Meta, error)

	// Update changes metadata for an entry.
	Update(meta Meta) error
	// Delete
	// Create
}

// Meta is the interface for a single entry (a nodebook or folder) in a
// Repository.
// These entries are used to access and change metadata for an item.
//
// The Reader() method can be used to download additional content, i.e. the
// pages and drawings for a notebook.
type Meta interface {
	ID() string
	Version() uint
	Name() string
	SetName(n string)
	Type() NotebookType
	Pinned() bool
	SetPinned(p bool)
	LastModified() time.Time
	Parent() string
	// Reader creates a reader for one of the components associated with an
	// item, e.g. the drawing for a single page.
	//
	// This function is typically used internally by ReadDocument and friends.
	Reader(path ...string) (io.ReadCloser, error)
	// Writer()

	// PagePrefix returns the filename prefix for page related paths.
	//
	// This function is normally used internally by ReadDocument and friends.
	PagePrefix(pageID string, pageIndex int) string
}

// ReadDocument is a helper function to read a full Document from a repository entry.
func ReadDocument(m Meta) (*Document, error) {
	if m.Type() != DocumentType {
		return nil, fmt.Errorf("can opnly read document for items with type DocumentType")
	}

	cp := m.ID() + ".content"
	cr, err := m.Reader(cp)
	if err != nil {
		return nil, err
	}
	defer cr.Close()

	var c Content
	err = json.NewDecoder(cr).Decode(&c)
	if err != nil {
		return nil, err
	}

	return &Document{
		Meta:    m,
		content: &c,
	}, nil
}

// A Document is a notebook, PDF or EPUB with all associated metadata
// and Drawings.
//
// A Document is internally backed by a Repository and can load additional
// content as it is requested.
type Document struct {
	Meta
	content  *Content
	pagedata []Pagedata
	pages    map[string]*Page
}

func NewDocument(ft FileType) *Document {
	return &Document{
		content:  NewContent(ft),
		pagedata: make([]Pagedata, 0),
	}
}

func (d *Document) Validate() error {
	err := d.content.Validate()
	if err != nil {
		return err
	}

	return nil
}

// PageCount returns the number of pages in this documents.
//
// Note that for PDF and EPUB files, the number of drawings can be less than
// the number of pages.
func (d *Document) PageCount() int {
	return d.content.PageCount
}

// Pages returns a list of page IDs on the correct oreder.
func (d *Document) Pages() []string {
	return d.content.Pages
}

// FileType is one of the supported types of content (Notebook, PDF, EPUB).
func (d *Document) FileType() FileType {
	return d.content.FileType
}

// Orientation is the base layout (Portait or Landscape) for this document.
func (d *Document) Orientation() Orientation {
	return d.content.Orientation
}

// CoverPage is the number of the page that should be used as a cover.
func (d *Document) CoverPage() int {
	// fallback on lastOpenedPage ?
	return d.content.CoverPageNumber
}

// Page loads meta data associated with the given pageID.
func (d *Document) Page(pageID string) (*Page, error) {
	if d.pages != nil {
		p := d.pages[pageID]
		if p != nil {
			return p, nil
		}
	}

	idx, err := d.pageIndex(pageID)
	if err != nil {
		return nil, err
	}

	// lazy load pagedata
	if d.pagedata == nil {
		pdp := d.ID() + ".pagedata"
		pdr, err := d.Reader(pdp)
		if err != nil {
			return nil, err
		}
		defer pdr.Close()
		pd, err := ReadPagedata(pdr)
		if err != nil {
			return nil, err
		}
		d.pagedata = pd
	}

	// check if we have pagedata for this page
	// TODO: we might set a default if we have none
	if len(d.pagedata) <= idx {
		return nil, fmt.Errorf("no pagedata for page with id %q", pageID)
	}

	// Load page metadata
	var pm PageMetadata
	pmp := d.PagePrefix(d.ID(), idx) + "-metadata.json"
	pmr, err := d.Reader(d.ID(), pmp)
	if err != nil {
		logging.Debug("No page metadata for page %v at %q", idx, pmp)
		// xxx-metadata.json seems to be optional.
		// Probably(?) the last (empty) page in a notebook has no metadata
		// check if this is a NotFoundError
		notFound := true
		if !notFound {
			return nil, err
		}
	} else {
		err = json.NewDecoder(pmr).Decode(&pm)
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		if pmr != nil {
			pmr.Close()
		}
	}()

	// construct the Page item
	p := &Page{
		index:    idx,
		meta:     pm,
		pagedata: d.pagedata[idx],
	}

	// cache
	if d.pages == nil {
		d.pages = make(map[string]*Page)
	}
	d.pages[pageID] = p

	return p, nil
}

// Drawing loads the handwritten drawing for the given pageID.
//
// Note that not all pages have associated drawings.
// If a page has no drawing...
// TODO: return a specific type of error
func (d *Document) Drawing(pageID string) (*Drawing, error) {
	idx, err := d.pageIndex(pageID)
	if err != nil {
		return nil, err
	}

	dp := d.PagePrefix(d.ID(), idx) + ".rm"
	dr, err := d.Reader(d.ID(), dp)
	if err != nil {
		return nil, err
	}
	defer dr.Close()

	drawing, err := ReadDrawing(dr)
	if err != nil {
		return nil, err
	}

	return drawing, nil
}

// AttachmentReader returns a reader for an associated PDF or EPUB files
// according to FileType().
//
// An error is returned if this document has no associated attachment.
func (d *Document) AttachmentReader() (io.ReadCloser, error) {
	p := d.ID()
	switch d.FileType() {
	case Pdf:
		p += ".pdf"
	case Epub:
		p += ".epub"
	default:
		return nil, fmt.Errorf("document of type %v has no attachment", d.FileType())
	}

	return d.Reader(p)
}

func (d *Document) pageIndex(pageID string) (int, error) {
	// Check if that page id exists
	// AND determine the page index
	for i, id := range d.Pages() {
		if id == pageID {
			return i, nil
		}
	}

	return 0, fmt.Errorf("invalid page id %q", pageID)
}

// Page describes a single page within a document.
type Page struct {
	index    int
	meta     PageMetadata
	pagedata Pagedata
}

// Number is the 1-based page number.
func (p *Page) Number() uint {
	return uint(p.index + 1)
}

// Orientation is the layout orientation for this specific page.
// It refers to the orientation of the background template.
func (p *Page) Orientation() Orientation {
	return p.pagedata.Orientation
}

// Template is the name of the background template.
// It can be used to look up a graphic file for this template.
func (p *Page) Template() string {
	return p.pagedata.Text
}

// HasTemplate tells if this page is associated with a background template.
// Returns false for the "Blank" template.
func (p *Page) HasTemplate() bool {
	return p.pagedata.HasTemplate()
}

// Layers is the metadata for the layers in this page.
func (p *Page) Layers() []LayerMetadata {
	if p.meta.Layers == nil {
		p.meta.Layers = make([]LayerMetadata, 0)
	}
	return p.meta.Layers
}
