package rm

import (
	"errors"
	"testing"
)

func TestNewDocument(t *testing.T) {
	//repo := &testRepo{}
	d := NewDocument("my document", Notebook)
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
