package rm

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Repository is the interface for the storage backend.
//
// It can either represent local files copied from the tablet
// or notes accessed via the Cloud API.
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
// these entries are used to access and change metadata for an item.
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
	// This function is normally used internally by ReadDocument and friends.
	Reader(path ...string) (io.ReadCloser, error)
	// Writer()
}

func ReadDocument(m Meta, kind string) (*Document, error) {
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
		kind:    kind,
		content: &c,
	}, nil
}

type Document struct {
	Meta
	kind     string
	content  *Content
	pagedata []Pagedata
	pages    map[string]*Page
}

func (d *Document) PageCount() uint {
	return d.content.PageCount
}

func (d *Document) Pages() []string {
	return d.content.Pages
}

func (d *Document) FileType() string {
	return d.content.FileType
}

func (d *Document) Orientation() string {
	return d.content.Orientation
}

func (d *Document) Page(pageId string) (*Page, error) {
	if d.pages != nil {
		p := d.pages[pageId]
		if p != nil {
			return p, nil
		}
	}

	idx, err := d.pageIndex(pageId)
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
		return nil, fmt.Errorf("no pagedata for page with id %q", pageId)
	}

	// Load page metadata
	prefix, err := d.pagePrefix(pageId, idx)
	if err != nil {
		return nil, err
	}
	pmp := prefix + "-metadata.json"

	var pm PageMetadata
	pmr, err := d.Reader(d.ID(), pmp)
	if err != nil {
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
	d.pages[pageId] = p

	return p, nil
}

func (d *Document) Drawing(pageId string) (*Drawing, error) {
	idx, err := d.pageIndex(pageId)
	if err != nil {
		return nil, err
	}

	// Load page metadata
	prefix, err := d.pagePrefix(pageId, idx)
	if err != nil {
		return nil, err
	}
	dp := prefix + ".rm"
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

func (d *Document) pageIndex(pageId string) (int, error) {
	// Check if that page id exists
	// AND determine the page index
	for i, id := range d.Pages() {
		if id == pageId {
			return i, nil
		}
	}

	return 0, fmt.Errorf("invalid page id %q", pageId)
}

func (d *Document) pagePrefix(pageId string, pageIndex int) (string, error) {
	// Depending on the repository type, the path is different:
	// - Filesystem: pageId
	// - API: page index
	switch d.kind {
	case "filesystem":
		return pageId, nil
	case "api":
		return fmt.Sprintf("%d", pageIndex), nil
	default:
		return "", fmt.Errorf("invalid repository kind %q", d.kind)
	}
}

func (p *Document) HasDrawing(pageId string) bool {
	// TODO: How do we find out?
	return true
}

type Page struct {
	index    int
	meta     PageMetadata
	pagedata Pagedata
}

func (p *Page) Number() uint {
	return uint(p.index + 1)
}

func (p *Page) Layout() PageLayout {
	return p.pagedata.Layout
}

func (p *Page) Template() string {
	return p.pagedata.Text
}

func (p *Page) HasTemplate() bool {
	return p.pagedata.HasTemplate()
}

func (p *Page) Layers() []LayerMetadata {
	if p.meta.Layers == nil {
		p.meta.Layers = make([]LayerMetadata, 0)
	}
	return p.meta.Layers
}
