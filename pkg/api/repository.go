package api

import (
	"time"

	"akeil.net/akeil/rm"
)

type repo struct {
	client *Client
}

func NewRepository(c *Client) rm.Repository {
	return &repo{
		client: c,
	}
}

func (r *repo) List() ([]rm.Meta, error) {
	items, err := r.client.List()
	if r != nil {
		return nil, err
	}

	rv := make([]rm.Meta, len(items))
	for i, item := range items {
		rv[i] = metaWrapper{item}
	}

	return rv, nil
}

func (r *repo) Fetch(id string) (rm.Meta, error) {
	item, err := r.client.Fetch(id)
	if err != nil {
		return nil, err
	}

	return metaWrapper{item}, nil
}

func (r *repo) Update(m rm.Meta) error {
	item := Item{
		ID:          m.ID(),
		Version:     int(m.Version()),
		Type:        m.Type(),
		VisibleName: m.Name(),
		Bookmarked:  m.Pinned(),
		Parent:      m.Parent(),
	}
	return r.client.update(item)
}

// implement the Meta interface for an Item
type metaWrapper struct {
	i Item
}

func (m metaWrapper) ID() string {
	return m.i.ID
}

func (m metaWrapper) Version() uint {
	return uint(m.i.Version)
}

func (m metaWrapper) Name() string {
	return m.i.VisibleName
}

func (m metaWrapper) SetName(n string) {
	m.i.VisibleName = n
}

func (m metaWrapper) Type() rm.NotebookType {
	//return m.i.Type
	// TODO:
	return rm.DocumentType
}

func (m metaWrapper) Pinned() bool {
	return m.i.Bookmarked
}

func (m metaWrapper) SetPinned(b bool) {
	m.i.Bookmarked = b
}

func (m metaWrapper) LastModified() time.Time {
	return m.i.ModifiedClient.Time
}

func (m metaWrapper) Parent() string {
	return m.i.Parent
}
