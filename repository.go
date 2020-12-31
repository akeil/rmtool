package rm

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	"akeil.net/akeil/rm/internal/logging"
)

type WriterFunc func(path ...string) (io.WriteCloser, error)

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

	// Reader creates a reader for one of the components associated with an
	// item, e.g. the drawing for a single page.
	//
	// This function is typically used internally by ReadDocument and friends.
	Reader(id string, version uint, path ...string) (io.ReadCloser, error)
	// Writer()

	// PagePrefix returns the filename prefix for page related paths.
	//
	// This function is normally used internally by ReadDocument and friends.
	PagePrefix(pageID string, pageIndex int) string

	Upload(d *Document) error
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
	// TODO: SetParent() ?

	// Validate checks the internal state of this item
	// and returns an error if it is not valid.
	Validate() error
}

// ReadDocument is a helper function to read a full Document from a repository entry.
func ReadDocument(r Repository, m Meta) (*Document, error) {
	if m.Type() != DocumentType {
		return nil, fmt.Errorf("can only read document for items with type DocumentType")
	}

	cp := m.ID() + ".content"
	logging.Debug("Read content info from %q", cp)
	cr, err := r.Reader(m.ID(), m.Version(), cp)
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
		repo:    r,
	}, nil
}

// A Document is a notebook, PDF or EPUB with all associated metadata
// and Drawings.
//
// A Document is internally backed by a Repository and can load additional
// content as it is requested.
type Document struct {
	Meta
	content    *Content
	pagedata   []Pagedata
	pages      map[string]*Page
	pagesMx    sync.Mutex
	drawings   map[string]*Drawing
	drawingsMx sync.Mutex
	repo       Repository
}

func NewDocument(name string, ft FileType) *Document {
	return &Document{
		Meta:     newDocMeta(DocumentType, name),
		content:  NewContent(ft),
		pagedata: make([]Pagedata, 0),
	}
}

// TODO: template name?
func NewNotebook(name string) *Document {
	d := NewDocument(name, Notebook)
	// new notbeooks are created with an empty first page
	d.CreatePage()
	return d
}

// TODO - implement
func NewPdf(name string, r io.ReadCloser) *Document {
	return NewDocument(name, Pdf)
}

// TODO - implement
func NewEpub(name string, r io.ReadCloser) *Document {
	return NewDocument(name, Epub)
}

func (d *Document) Validate() error {
	err := d.Meta.Validate()
	if err != nil {
		return err
	}

	if d.Meta.Type() != DocumentType {
		return NewValidationError("only DocumentType allowed, found %q", d.Meta.Type())
	}

	err = d.content.Validate()
	if err != nil {
		return err
	}

	for _, pd := range d.pagedata {
		err = pd.Validate()
		if err != nil {
			return err
		}
	}

	// len pagedata must match the number of pages
	if len(d.pagedata) != d.PageCount() {
		return NewValidationError("number of pagedata entries does not match page count: %v != %v", len(d.pagedata), d.PageCount())
	}

	// TODO: Notebook needs a drawing for each page
	// and at least one page

	// TODO: validate pages

	return nil
}

func (d *Document) Write(repo Repository, w WriterFunc) error {
	logging.Debug("write content")
	cw, err := w(fmt.Sprintf("%v.content", d.ID()))
	if err != nil {
		return err
	}
	err = json.NewEncoder(cw).Encode(d.content)
	if err != nil {
		return err
	}
	defer cw.Close()

	logging.Debug("Write pagedata")
	pw, err := w(fmt.Sprintf("%v.pagedata", d.ID()))
	if err != nil {
		return err
	}
	err = WritePagedata(d.pagedata, pw)
	if err != nil {
		return err
	}
	defer pw.Close()

	for i, pageID := range d.Pages() {
		logging.Debug("Write page metadata for %v", pageID)
		// TODO relies on all pages being cached
		p, err := d.Page(pageID)
		if err != nil {
			return err
		}
		prefix := repo.PagePrefix(pageID, i)
		pmw, err := w(d.ID(), prefix+"-metadata.json")
		if err != nil {
			return err
		}
		defer pmw.Close()
		err = json.NewEncoder(pmw).Encode(p.meta)
		if err != nil {
			return err
		}

		// we do not have a backing repository and can only write cached drawing
		// TODO: this does not feel like the "right" way to do it
		logging.Debug("Write drawing for %v", pageID)
		dr, err := d.Drawing(pageID)
		if err != nil {
			return err
		}
		drw, err := w(d.ID(), prefix+".rm")
		if err != nil {
			return err
		}
		defer drw.Close()
		err = WriteDrawing(drw, dr)
		if err != nil {
			return err
		}
	}

	// TODO: write attached PDF or EPUB

	// TODO write thumbnails?

	return nil
}

// Create a new page with a drawing and append it to the document.
// TODO: Orientation? Template?
func (d *Document) CreatePage() *Page {
	d.pagesMx.Lock()
	defer d.pagesMx.Unlock()

	pageID := uuid.New().String()

	d.content.Pages = append(d.content.Pages, pageID)
	d.content.PageCount++

	index := len(d.pagedata) // we'll append later, so index == size

	// with default orientation and default template
	pgData := newPagedata()
	d.pagedata = append(d.pagedata, pgData)

	pgMeta := PageMetadata{
		Layers: []LayerMetadata{
			LayerMetadata{
				Name: "Layer 1",
			},
		},
	}

	p := &Page{
		index:    index,
		meta:     pgMeta,
		pagedata: pgData,
	}

	// page cache
	if d.pages == nil {
		d.pages = make(map[string]*Page)
	}
	d.pages[pageID] = p

	// drawing
	d.drawingsMx.Lock()
	defer d.drawingsMx.Unlock()
	if d.drawings == nil {
		d.drawings = make(map[string]*Drawing)
	}
	d.drawings[pageID] = NewDrawing()

	return p
}

// PageCount returns the number of pages in this document.
//
// Note that for PDF and EPUB files, the number of drawings can be less than
// the number of pages.
func (d *Document) PageCount() int {
	return d.content.PageCount
}

// Pages returns a list of page IDs on the correct order.
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
	d.pagesMx.Lock()
	defer d.pagesMx.Unlock()

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

	// lazy load pagedata, guarded by pagesMx
	if d.pagedata == nil {
		pdp := d.ID() + ".pagedata"
		logging.Debug("Read pagedata from %q", pdp)
		pdr, err := d.reader(pdp)
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
	if len(d.pagedata) <= idx {
		return nil, fmt.Errorf("no pagedata for page with id %q", pageID)
	}

	// Load page metadata
	var pm PageMetadata
	pmp := d.repo.PagePrefix(d.ID(), idx) + "-metadata.json"
	logging.Debug("Read page metadata from %q", pmp)
	pmr, err := d.reader(d.ID(), pmp)
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
// If a page has no drawing, an error of type "Not Found" is returned
// (use IsNotFound(err) to check for this).
func (d *Document) Drawing(pageID string) (*Drawing, error) {
	d.drawingsMx.Lock()
	defer d.drawingsMx.Unlock()

	if d.drawings == nil {
		d.drawings = make(map[string]*Drawing)
	}
	cached := d.drawings[pageID]
	if cached != nil {
		return cached, nil
	}

	idx, err := d.pageIndex(pageID)
	if err != nil {
		return nil, err
	}

	dp := d.repo.PagePrefix(d.ID(), idx) + ".rm"
	logging.Debug("Read drawing from %q", dp)
	dr, err := d.reader(d.ID(), dp)
	if err != nil {
		return nil, err
	}
	defer dr.Close()

	drawing, err := ReadDrawing(dr)
	if err != nil {
		return nil, err
	}

	d.drawings[pageID] = drawing

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

	logging.Debug("Read attachment from %q", p)
	return d.reader(p)
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

func (d *Document) reader(path ...string) (io.ReadCloser, error) {
	return d.repo.Reader(d.ID(), d.Version(), path...)
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

// TODO: set orientation

// Template is the name of the background template.
// It can be used to look up a graphic file for this template.
func (p *Page) Template() string {
	return p.pagedata.Text
}

// TODO set template

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

// docMeta is used to hold metadata for newly created documents.
type docMeta struct {
	id           string
	version      uint
	nbType       NotebookType
	name         string
	pinned       bool
	lastModified time.Time
	parent       string
}

func newDocMeta(t NotebookType, name string) Meta {
	return &docMeta{
		id:           uuid.New().String(),
		nbType:       t,
		name:         name,
		lastModified: time.Now(),
	}
}

func (d *docMeta) ID() string {
	return d.id
}

func (d *docMeta) Version() uint {
	return d.version
}

func (d *docMeta) Name() string {
	return d.name
}

func (d *docMeta) SetName(n string) {
	d.name = n
}

func (d *docMeta) Type() NotebookType {
	return d.nbType
}

func (d *docMeta) Pinned() bool {
	return d.pinned
}

func (d *docMeta) SetPinned(p bool) {
	d.pinned = p
}

func (d *docMeta) LastModified() time.Time {
	return d.lastModified
}

func (d *docMeta) Parent() string {
	return d.parent
}

func (d *docMeta) Reader(path ...string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *docMeta) PagePrefix(pageID string, pageIndex int) string {
	return pageID
}

func (d *docMeta) Validate() error {
	switch d.Type() {
	case DocumentType, CollectionType:
		// ok
	default:
		return NewValidationError("invalid type %v", d.Type())
	}

	if d.Name() == "" {
		return NewValidationError("name must not be empty")
	}

	return nil
}
