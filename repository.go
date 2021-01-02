package rm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"

	"github.com/akeil/rm/internal/errors"
	"github.com/akeil/rm/internal/logging"
)

type WriterFunc func(path ...string) (io.WriteCloser, error)

// An AttachmentReader creates a reader for a PDF or EPUB attachment.
//
// It must be supplied when creating documents with attachments.
// It must be possible to call this function multiple times.
type AttachmentReader func() (io.ReadCloser, error)

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
	// TODO Delete
	// TODO Create

	// TODO CreateFolder ?
	// TODO DeleteFolder ?

	// Reader creates a reader for one of the components associated with an
	// item, e.g. the drawing for a single page.
	//
	// This function is typically used internally by ReadDocument and friends.
	Reader(id string, version uint, path ...string) (io.ReadCloser, error)

	// PagePrefix returns the filename prefix for page related paths.
	//
	// This function is normally used internally by ReadDocument and friends.
	PagePrefix(pageID string, pageIndex int) string

	// Upload creates the given document in the repository.
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

	// Validate checks the internal state of this item
	// and returns an error if it is not valid.
	Validate() error
}

// ReadDocument is a helper function to read a full Document from a repository entry.
// TODO make this a method of the repository, transfer implementation to internal/
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

// Page describes a single page within a document.
type Page struct {
	index    int
	meta     *PageMetadata
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
	if p.meta == nil || p.meta.Layers == nil {
		return make([]LayerMetadata, 0)
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

func newDocMeta(t NotebookType, name, parentID string) Meta {
	return &docMeta{
		id:           uuid.New().String(),
		nbType:       t,
		name:         name,
		parent:       parentID,
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
		return errors.NewValidationError("invalid type %v", d.Type())
	}

	if d.Name() == "" {
		return errors.NewValidationError("name must not be empty")
	}

	return nil
}

// PDF Helper -----------------------------------------------------------------

func countPdfPages(rc io.ReadCloser) (int, error) {
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return 0, err
	}
	rc.Close()
	rs := io.ReadSeeker(bytes.NewReader(data))

	cfg := pdfcpu.NewDefaultConfiguration()
	ctx, err := pdfcpu.Read(rs, cfg)
	if err != nil {
		return 0, err
	}

	// This *must* be called before accessing page count
	err = ctx.EnsurePageCount()
	if err != nil {
		return 0, err
	}

	return ctx.PageCount, nil
}

// EPUB Helper ----------------------------------------------------------------

func countEpubPages(rc io.ReadCloser) (int, error) {
	// TODO: find a library to do this
	// - https://github.com/bmaupin/go-epub  --  creates EPUB
	// - https://github.com/kapmahc/epub	 --  reads from file path only
	//
	// ... does it even make sense?
	// One can change font size, line height, margins in the reader.
	// -> The number of pages depends on individual settings.
	return 0, fmt.Errorf("not implemented")
}
