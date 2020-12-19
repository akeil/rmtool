package rm

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type fsStorage struct {
	Base string
}

// Creates a storage that is based on the directory structure as found on the
// tablet itself.
func NewFilesystemStorage(base string) Storage {
	return &fsStorage{base}
}

func (f *fsStorage) List() ([]string, error) {
	files, err := ioutil.ReadDir(f.Base)
	if err != nil {
		return nil, err
	}

	l := make([]string, 0)
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".metadata" {
			l = append(l, strings.TrimSuffix(file.Name(), ".metadata"))
		}
	}

	return l, err
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

func (f *fsStorage) HasDrawing(id, pageId string) (bool, error) {
	path := filepath.Join(f.Base, id, pageId+".rm")
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
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
