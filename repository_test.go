package rm

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"time"
)

func TestNewDocument(t *testing.T) {
	repo := &testRepo{}
	m := repo.NewDocumentMeta()

	d := NewDocument(m, Notebook)
	err := d.Validate()
	if err != nil {
		t.Log("newly created document is not valid")
		t.Error(err)
	}
}

// Test helpers ---------------------------------------------------------------

type testRepo struct{}

func (r *testRepo) List() ([]Meta, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepo) Update(meta Meta) error {
	return errors.New("not implemented")
}

func (r *testRepo) NewDocumentMeta() Meta {
	return &testMeta{
		repository: r,
		id:         "generated",
		version:    0,
		nbType:     DocumentType,
	}
}

type testMeta struct {
	repository   Repository
	id           string
	version      uint
	nbType       NotebookType
	name         string
	pinned       bool
	lastModified time.Time
	parent       string
}

func (t *testMeta) ID() string {
	return t.id
}

func (t *testMeta) Version() uint {
	return t.version
}

func (t *testMeta) Name() string {
	return t.name
}

func (t *testMeta) SetName(n string) {
	t.name = n
}

func (t *testMeta) Type() NotebookType {
	return t.nbType
}

func (t *testMeta) Pinned() bool {
	return t.pinned
}
func (t *testMeta) SetPinned(p bool) {
	t.pinned = p
}
func (t *testMeta) LastModified() time.Time {
	return t.lastModified
}

func (t *testMeta) Parent() string {
	return t.parent
}

func (t *testMeta) Reader(path ...string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func (t *testMeta) PagePrefix(pageID string, pageIndex int) string {
	return pageID
}

func (t *testMeta) Validate() error {
	return nil
}
