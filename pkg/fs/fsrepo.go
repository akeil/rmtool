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
)

type repo struct {
	base string
}

func NewRepository(base string) rm.Repository {
	return &repo{
		base: base,
	}
}

func (r *repo) List() ([]rm.Meta, error) {
	fmt.Printf("List files from %q\n", r.base)

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
			l = append(l, metaWrapper{id: id, i: &m})
		}
	}

	return l, err
}

func (r *repo) Update(m rm.Meta) error {
	p := filepath.Join(r.base, m.ID()+".metadata")
	o, err := readMetadata(p)
	if err != nil {
		return err
	}

	// check the version
	if m.Version() != o.Version {
		return fmt.Errorf("version mismatch %d != %d", m.Version(), o.Version)
	}

	o.Version += 1
	o.LastModified = rm.Timestamp{time.Now()}

	// assumption: we need to set these if we write to the tablet
	o.Synced = false
	o.Metadatamodified = true

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

	err = json.NewEncoder(f).Encode(&o)
	if err != nil {
		return err
	}

	fmt.Printf("Move updated JSON document to %q\n", p)

	return os.Rename(f.Name(), p)
}

func (r *repo) reader(id string, path ...string) (io.ReadCloser, error) {
	parts := []string{r.base}
	parts = append(parts, path...)
	p := filepath.Join(parts...)

	fmt.Printf("Create reader for %q\n", p)

	return os.Open(p)
}

func readMetadata(path string) (rm.Metadata, error) {
	var m rm.Metadata
	r, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer r.Close()

	err = json.NewDecoder(r).Decode(&m)

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
