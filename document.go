package rmtool

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"

	"github.com/akeil/rmtool/internal/errors"
	"github.com/akeil/rmtool/internal/logging"
	"github.com/akeil/rmtool/pkg/lines"
)

// A Document is a notebook, PDF or EPUB with all associated metadata
// and Drawings.
//
// A Document is internally backed by a Repository and can load additional
// content as it is requested.
type Document struct {
	Meta
	content          *Content
	pagedata         []Pagedata
	pages            map[string]*Page
	pagesMx          sync.Mutex
	drawings         map[string]*lines.Drawing
	drawingsMx       sync.Mutex
	attachmentReader AttachmentReader
	repo             Repository
}

// NewNotebook creates a new document of type "notebook" with a single emtpty page.
// TODO: template name?
func NewNotebook(name, parentID string) *Document {
	d := newDocument(name, parentID, Notebook, nil)
	// new notbeooks are created with an empty first page
	d.CreatePage()
	return d
}

// NewPdf creates a new document for a PDF file.
//
// The given AttachmentReader should return a Reader for the PDF file.
// Note that this can return an error as the PDF needs to be read for this.
func NewPdf(name, parentID string, r AttachmentReader) (*Document, error) {
	d := newDocument(name, parentID, Pdf, r)
	err := d.createPdfPages()
	// TODO: orientation of the document?
	return d, err
}

// TODO - implement
func NewEpub(name, parentID string, r AttachmentReader) *Document {
	return newDocument(name, parentID, Epub, r)
}

func newDocument(name, parentID string, ft FileType, r AttachmentReader) *Document {
	return &Document{
		Meta:             newDocMeta(DocumentType, name, parentID),
		content:          NewContent(ft),
		pagedata:         make([]Pagedata, 0),
		attachmentReader: r,
	}
}

func (d *Document) Validate() error {
	err := d.Meta.Validate()
	if err != nil {
		return err
	}

	if d.Meta.Type() != DocumentType {
		return errors.NewValidationError("only DocumentType allowed, found %q", d.Meta.Type())
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
		return errors.NewValidationError("number of pagedata entries does not match page count: %v != %v", len(d.pagedata), d.PageCount())
	}

	switch d.FileType() {
	case Notebook:
		err = d.validateNotebook()
	case Pdf:
		err = d.validateAttachment()
	case Epub:
		err = d.validateAttachment()
	}
	if err != nil {
		return err
	}

	// TODO: validate pages

	return nil
}

func (d *Document) validateNotebook() error {
	// Notebook needs a drawing for each page and at least one page
	if d.PageCount() < 1 {
		return errors.NewValidationError("notbeook must have at least one page")
	}
	// TODO: checking only cached drawings means we can validate
	// fully loaded or new notebooks only
	for _, pageID := range d.Pages() {
		dr := d.drawings[pageID]
		if dr == nil {
			return errors.NewValidationError("page %q has no associated drawing", pageID)
		}
		err := dr.Validate()
		if err != nil {
			return err
		}

		// TODO must have PageMetadata with at least one layer
	}

	return nil
}

// for PDF or EPUB
func (d *Document) validateAttachment() error {
	if d.attachmentReader == nil {
		return errors.NewValidationError("missing attachment reader")
	}
	// TODO - do we need more validation?

	return nil
}

func (d *Document) Write(repo Repository, w WriterFunc) error {
	// .content and .pagedata
	err := d.writeContent(w)
	if err != nil {
		return err
	}

	// page meta and drawings
	err = d.writePages(repo, w)
	if err != nil {
		return err
	}

	// attached PDF or EPUB
	if d.FileType() == Pdf || d.FileType() == Epub {
		err = d.writeAttachment(w)
		if err != nil {
			return nil
		}
	}

	// TODO write thumbnails?

	return nil
}

// writes the .content and the .pagedata files.
func (d *Document) writeContent(w WriterFunc) error {
	logging.Debug("Write content")
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

	return nil
}

// writes the drawings (.rm) and the metadata for each page that has a drawing.
// writes nothing for pages w/o drawing
func (d *Document) writePages(repo Repository, w WriterFunc) error {
	d.pagesMx.Lock()
	d.drawingsMx.Lock()
	defer d.pagesMx.Unlock()
	defer d.drawingsMx.Unlock()

	for i, pageID := range d.Pages() {
		// we do not have a backing repository and can only write cached drawing
		// TODO: this does not feel like the "right" way to do it
		dr := d.drawings[pageID]
		if dr == nil {
			logging.Debug("Page %q has no drawing", pageID)
			continue
		}

		// TODO relies on all pages being cached
		logging.Debug("Write page metadata for %v", pageID)
		p := d.pages[pageID]
		if p == nil {
			return fmt.Errorf("missing page metadata for page %q", pageID)
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

		logging.Debug("Write drawing for %v", pageID)
		drw, err := w(d.ID(), prefix+".rm")
		if err != nil {
			return err
		}
		defer drw.Close()
		err = lines.WriteDrawing(drw, dr)
		if err != nil {
			return err
		}
	}

	return nil
}

// write attachment, assume FileType is Pdf or Epub
func (d *Document) writeAttachment(w WriterFunc) error {
	logging.Debug("Write attachment (type=%v)", d.FileType())
	if d.attachmentReader == nil {
		return fmt.Errorf("missing attachment reader")
	}

	path := d.ID() + d.FileType().Ext()
	aw, err := w(path)
	if err != nil {
		return err
	}
	defer aw.Close()
	ar, err := d.attachmentReader()
	if err != nil {
		return err
	}
	defer ar.Close()
	_, err = io.Copy(aw, ar)
	if err != nil {
		return err
	}

	return nil
}

// CreatePage creates a new page with a drawing and append it to the document.
// TODO: Orientation? Template?
func (d *Document) CreatePage() string {
	pgMeta := &PageMetadata{
		Layers: []LayerMetadata{
			LayerMetadata{
				Name: "Layer 1",
			},
		},
	}
	pageID := d.addPage(pgMeta)

	// drawing
	d.drawingsMx.Lock()
	defer d.drawingsMx.Unlock()
	if d.drawings == nil {
		d.drawings = make(map[string]*lines.Drawing)
	}
	d.drawings[pageID] = lines.NewDrawing()

	return pageID
}

func (d *Document) createPdfPages() error {
	rc, err := d.attachmentReader()
	if err != nil {
		return err
	}

	numPages, err := countPdfPages(rc)
	if err != nil {
		return err
	}

	for i := 0; i < numPages; i++ {
		d.addPage(nil)
	}

	return nil
}

// adds an empty page WITHOUT drawing
func (d *Document) addPage(pgMeta *PageMetadata) string {
	d.pagesMx.Lock()
	defer d.pagesMx.Unlock()

	pageID := uuid.New().String()

	d.content.Pages = append(d.content.Pages, pageID)
	d.content.PageCount++

	index := len(d.pagedata) // we'll append later, so index == size

	// with default orientation and default template
	pgData := newPagedata()
	d.pagedata = append(d.pagedata, pgData)

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

	return pageID
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
	pm := &PageMetadata{}
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
		err = json.NewDecoder(pmr).Decode(pm)
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
func (d *Document) Drawing(pageID string) (*lines.Drawing, error) {
	d.drawingsMx.Lock()
	defer d.drawingsMx.Unlock()

	if d.drawings == nil {
		d.drawings = make(map[string]*lines.Drawing)
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

	drawing, err := lines.ReadDrawing(dr)
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
