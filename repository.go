package rm

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Repository interface {
	List() ([]Meta, error)

	Fetch(id string) (Meta, error)
	Update(meta Meta) error

	//Get(id string, version uint) (Document, error)

	// Put(d Document) error
}

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
}

type storageAdapter interface {
	Read(id string, version uint, relPath ...string) (io.ReadCloser, error)
}

func readDocument(m Meta, adapter storageAdapter) (*Document, error) {
	cp := m.ID() + ".content"
	r, err := adapter.Read(m.ID(), m.Version(), cp)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var c Content
	err = json.NewDecoder(r).Decode(&c)
	if err != nil {
		return nil, err
	}

	return &Document{
		adapter: adapter,
		meta:    m,
	}, nil
}

type Document struct {
	adapter  storageAdapter
	meta     Meta
	content  *Content
	pagedata []Pagedata
	pages    map[string]*PageX
}

func (d *Document) ID() string {
	return d.meta.ID()
}

func (d *Document) Version() uint {
	return d.meta.Version()
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

func (d *Document) Drawing(pageId string) (*Drawing, error) {
	dp := pageId + ".rm"
	dr, err := d.adapter.Read(d.ID(), d.Version(), d.ID(), dp)
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

func (p *Document) HasDrawing(pageId string) bool {
	// TODO: How do we find out?
	return true
}

func (d *Document) Page(pageId string) (*PageX, error) {
	if d.pages != nil {
		p := d.pages[pageId]
		if p != nil {
			return p, nil
		}
	}

	// Check if that page id exists
	// AND determine the page number
	idx := -1
	for i, id := range d.Pages() {
		idx = i
		if id == pageId {
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("invalid page id %q", pageId)
	}

	// lazy load pagedata
	if d.pagedata == nil {
		pdp := d.ID() + ".pagedata"
		pdr, err := d.adapter.Read(d.ID(), d.Version(), pdp)
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

	// load page metadata
	pmp := pageId + "-metadata.json"
	pmr, err := d.adapter.Read(d.ID(), d.Version(), d.ID(), pmp)
	if err != nil {
		return nil, err
	}
	defer pmr.Close()

	var pm PageMetadata
	err = json.NewDecoder(pmr).Decode(&pm)
	if err != nil {
		return nil, err
	}

	// construct the Page item
	p := &PageX{
		index:    idx,
		meta:     pm,
		pagedata: d.pagedata[idx],
	}

	// cache
	if d.pages == nil {
		d.pages = make(map[string]*PageX)
	}
	d.pages[pageId] = p

	return p, nil
}

type PageX struct {
	index    int
	meta     PageMetadata
	pagedata Pagedata
}

func (p *PageX) Number() uint {
	return uint(p.index + 1)
}

func (p *PageX) Layout() PageLayout {
	return p.pagedata.Layout
}

func (p *PageX) TemplateName() string {
	return p.pagedata.Text
}

func (p *PageX) Layers() []LayerMetadata {
	return p.meta.Layers
}
