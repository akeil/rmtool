package api

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/akeil/rmtool"
	"github.com/akeil/rmtool/internal/errors"
	"github.com/akeil/rmtool/internal/logging"
)

type repo struct {
	client *Client
	cache  rmtool.Cache
}

// NewRepository creates a Repository with the reMarkable cloud service as
// backend.
//
// The supplied cache is used to store downloaded content (notebooks).
func NewRepository(c *Client, cache rmtool.Cache) rmtool.Repository {
	return &repo{
		client: c,
		cache:  cache,
	}
}

func (r *repo) List() ([]rmtool.Meta, error) {
	logging.Debug("Repository.List")
	items, err := r.client.List()
	if err != nil {
		return nil, err
	}

	rv := make([]rmtool.Meta, len(items))
	for i, item := range items {
		rv[i] = metaWrapper{i: item, r: r}
	}

	return rv, nil
}

func (r *repo) Update(m rmtool.Meta) error {
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

func (r *repo) PagePrefix(id string, index int) string {
	return fmt.Sprintf("%d", index)
}

func (r *repo) Reader(id string, version uint, path ...string) (io.ReadCloser, error) {
	// Data from cache or fresh download
	var data []byte
	data, err := r.fromCache(id, version)
	if err != nil {
		data, err = r.downloadAndCache(id, version)
		if err != nil {
			return nil, err
		}
	}

	br := bytes.NewReader(data)
	zr, err := zip.NewReader(br, int64(br.Len()))
	if err != nil {
		return nil, err
	}

	// Read the desired entry from the zip file
	match := strings.Join(path, "/")
	var entry *zip.File
	for _, zf := range zr.File {
		if zf.Name == match {
			entry = zf
			break
		}
	}
	if entry == nil {
		return nil, errors.NewNotFound("no zip entry found with name %q", match)
	}

	// Return a reader for the file entry
	return entry.Open()
}

func (r *repo) fromCache(id string, version uint) ([]byte, error) {
	cr, err := r.cache.Get(cacheKey(id, version))
	if err != nil {
		return nil, err
	}
	defer cr.Close()

	data, err := ioutil.ReadAll(cr)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *repo) downloadAndCache(id string, version uint) ([]byte, error) {
	i, err := r.client.fetchItem(id)
	if err != nil {
		return nil, err
	}

	logging.Debug("Download blob for %v.%v", id, version)

	var buf bytes.Buffer
	err = r.client.fetchBlob(i.BlobURLGet, &buf)
	if err != nil {
		return nil, err
	}

	// ignores cache errors
	r.cache.Put(cacheKey(id, version), bytes.NewReader(buf.Bytes()))

	// This should presumably remove the one outdated entry (if any).
	go r.cleanCache(id, version)

	logging.Debug("Buffer.Len() -> %v", buf.Len())
	logging.Debug("len(Buffer.Bytes()) -> %v", len(buf.Bytes()))

	return buf.Bytes(), nil
}

func (r *repo) Upload(d *rmtool.Document) error {
	err := d.Validate()
	if err != nil {
		return err
	}
	err = r.client.checkParent(d.Parent())
	if err != nil {
		return err
	}

	// Create the zip file for later upload
	buf := new(bytes.Buffer)
	archive := zip.NewWriter(buf)

	w := func(path ...string) (io.WriteCloser, error) {
		name := strings.Join(path, "/")
		logging.Debug("Create zip entry %q", name)
		writer, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		return &nopCloser{writer}, nil
	}

	logging.Debug("Write document parts to zip archive")
	err = d.Write(r, w)
	if err != nil {
		return err
	}

	err = archive.Close()
	if err != nil {
		return err
	}

	logging.Debug("Upload the zip archive")

	err = r.client.Upload(d.Name(), d.ID(), d.Parent(), buf)
	if err != nil {
		return err
	}

	return err
}

// cleanCache removes outdated versions from the cache.
func (r *repo) cleanCache(id string, version uint) {
	// TODO: not ideal, especially for high vversion numbers.
	// we'll blindly try to delete every entry except the current one,
	for i := uint(0); i < version; i++ {
		r.cache.Delete(cacheKey(id, i))
	}
}

func cacheKey(id string, version uint) string {
	return fmt.Sprintf("%v.%v.zip", id, version)
}

// implement the Meta interface for an Item
type metaWrapper struct {
	i Item
	r *repo
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

func (m metaWrapper) Type() rmtool.NotebookType {
	return m.i.Type
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

func (m metaWrapper) Validate() error {
	return m.i.Validate()
}

// implement empty Close for WriteCloser interface
type nopCloser struct {
	io.Writer
}

func (n *nopCloser) Close() error {
	return nil
}
