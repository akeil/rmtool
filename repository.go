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

	Reader(id string, version uint, path ...string) (io.ReadCloser, error)

	// Writer()
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

func ReadDocument(m Meta, repo Repository, kind string) (*Document, error) {
	cp := m.ID() + ".content"
	cr, err := repo.Reader(m.ID(), m.Version(), cp)
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
		repo:    repo,
		kind:    kind,
		meta:    m,
		content: &c,
	}, nil
}

type Document struct {
	repo     Repository
	kind     string
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
		pdr, err := d.repo.Reader(d.ID(), d.Version(), pdp)
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
	// Depending on the repository type, the path is different:
	// - Filesystem: pageId
	// - API: page index
	var prefix string
	switch d.kind {
	case "filesystem":
		prefix = pageId
	case "api":
		prefix = fmt.Sprintf("%d", idx)
	default:
		return nil, fmt.Errorf("invalid repository kind %q", d.kind)
	}
	pmp := prefix + "-metadata.json"

	var pm PageMetadata
	pmr, err := d.repo.Reader(d.ID(), d.Version(), d.ID(), pmp)
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

func (d *Document) Drawing(pageId string) (*Drawing, error) {
	dp := pageId + ".rm"
	dr, err := d.repo.Reader(d.ID(), d.Version(), d.ID(), dp)
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

func (p *PageX) Template() string {
	return p.pagedata.Text
}

func (p *PageX) Layers() []LayerMetadata {
	if p.meta.Layers == nil {
		p.meta.Layers = make([]LayerMetadata, 0)
	}
	return p.meta.Layers
}
