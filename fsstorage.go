package rm

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type fsStorage struct {
	Base string
}

// Creates a storage that is based on the directory structure as found on the
// tablet itself.
func NewFilesystemStorage(base string) Storage {
	return &fsStorage{base}
}

func (f *fsStorage) ReadMetadata(id string) (Metadata, error) {
	var m Metadata
	err := readJSON(f.Base, id+".metadata", &m)
	return m, err
}

func (f *fsStorage) ReadContent(id string) (Content, error) {
	var c Content
	err := readJSON(f.Base, id+".content", &c)
	return c, err
}

func (f *fsStorage) ReadPagedata(id string) ([]Pagedata, error) {
	path := filepath.Join(f.Base, id+".pagedata")
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ReadPagedata(r)
}

func (f *fsStorage) ReadDrawing(id, pageId string) (*Drawing, error) {
	path := filepath.Join(f.Base, id, pageId+".rm")
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ReadDrawing(r)
}

func readJSON(base, filename string, dst interface{}) error {
	path := filepath.Join(base, filename)
	r, err := os.Open(path)
	if err != nil {
		return err
	}
	defer r.Close()

	dec := json.NewDecoder(r)
	err = dec.Decode(dst)
	if err != nil {
		return err
	}

	return nil
}
