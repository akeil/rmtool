package fs

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/logging"
)

type repo struct {
	base string
}

// NewRepository creates a repository backed by the local file system.
//
// The given path should point to a directory similar to the storage directory
// on the remarkable tablet.
func NewRepository(path string) rm.Repository {
	return &repo{
		base: path,
	}
}

func (r *repo) List() ([]rm.Meta, error) {
	logging.Debug("List files from %q", r.base)

	files, err := ioutil.ReadDir(r.base)
	if err != nil {
		return nil, err
	}

	l := make([]rm.Meta, 0)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".metadata" {
			id := strings.TrimSuffix(f.Name(), ".metadata")
			p := filepath.Join(r.base, f.Name())
			m, err := readMetadata(p)
			if err != nil {
				return nil, err
			}
			l = append(l, metaWrapper{id: id, i: &m, repo: r})
		}
	}

	return l, err
}

func (r *repo) Update(m rm.Meta) error {
	logging.Debug("Update entry with id %q, version %v", m.ID(), m.Version())
	p := filepath.Join(r.base, m.ID()+".metadata")
	o, err := readMetadata(p)
	if err != nil {
		return err
	}

	// check the version
	if m.Version() != o.Version {
		return fmt.Errorf("version mismatch %d != %d", m.Version(), o.Version)
	}

	// TODO: check the parent

	o.Version++
	o.LastModified = rm.Timestamp{time.Now()}

	// assumption: we need to set these if we write to the tablet
	o.Synced = false
	o.MetadataModified = true

	// apply the changes
	o.VisibleName = m.Name()
	o.Pinned = m.Pinned()
	o.Parent = m.Parent()
	o.Type = m.Type()

	// to tempfile
	f, err := ioutil.TempFile("", "rm-*.json")
	if err != nil {
		return err
	}
	defer f.Close()

	logging.Debug("Write JSON to tempfile at %q", f.Name())
	err = json.NewEncoder(f).Encode(&o)
	if err != nil {
		return err
	}

	logging.Debug("Move updated JSON document to %q\n", p)

	return os.Rename(f.Name(), p)
}

func (r *repo) reader(id string, path ...string) (io.ReadCloser, error) {
	parts := []string{r.base}
	parts = append(parts, path...)
	p := filepath.Join(parts...)

	logging.Debug("Create reader for %q\n", p)

	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return f, rm.NewNotFound(err.Error())
	}
	return f, err
}

func readMetadata(path string) (rm.Metadata, error) {
	var m rm.Metadata
	r, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, rm.NewNotFound("no metadata file at %q", path)
		}
		return m, rm.Wrap(err, "failed to read metadata for %q", path)
	}
	defer r.Close()

	err = json.NewDecoder(r).Decode(&m)
	if err != nil {
		return m, rm.Wrap(err, "failed to read metadata for %q", path)
	}

	return m, err
}

type metaWrapper struct {
	id   string
	i    *rm.Metadata
	repo *repo
}

func (m metaWrapper) Reader(path ...string) (io.ReadCloser, error) {
	return m.repo.reader(m.ID(), path...)
}

func (m metaWrapper) ID() string {
	return m.id
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
	return m.i.Type
}

func (m metaWrapper) Pinned() bool {
	return m.i.Pinned
}

func (m metaWrapper) SetPinned(b bool) {
	m.i.Pinned = b
}

func (m metaWrapper) LastModified() time.Time {
	return m.i.LastModified.Time
}

func (m metaWrapper) Parent() string {
	return m.i.Parent
}

func (m metaWrapper) PagePrefix(id string, index int) string {
	return id
}
