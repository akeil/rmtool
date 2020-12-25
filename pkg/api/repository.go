package api

import (
	"archive/zip"
	"fmt"
	//    "os"
	"io"
	"io/ioutil"
	"strings"
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
	fmt.Println("repo.List")
	items, err := r.client.List()
	if err != nil {
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

func (r *repo) Reader(id string, version uint, path ...string) (io.ReadCloser, error) {
	// Retreive the BlobURLGet
	i, err := r.client.fetchItem(id)
	if err != nil {
		return nil, err
	}

	f, err := ioutil.TempFile("", "rm_"+id+"_*.zip")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fmt.Printf("Download blob to %q\n", f.Name())
	err = r.client.fetchBlob(i.BlobURLGet, f)
	if err != nil {
		return nil, err
	}

	// Read the desired entry from the zip file
	zr, err := zip.OpenReader(f.Name())
	if err != nil {
		return nil, err
	}
	//defer zr.Close()

	match := strings.Join(path, "/")
	var entry *zip.File
	for _, zf := range zr.File {
		fmt.Printf("Zip entry: %q\n", zf.Name)
		if zf.Name == match {
			entry = zf
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("no zip entry found with name %q", match)
	}
	// return a reader for the file
	// closing the reader should close the zip reader
	// and delete the tempfile
	return entry.Open()

	//return nil, nil
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
