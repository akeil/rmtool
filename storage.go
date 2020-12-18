package rm

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Storage interface {
	// ReadMetadata reads the metadata for a notebook with the given ID.
	ReadMetadata(id string) (Metadata, error)
	// ReadContent reads the content for a notebook.
	ReadContent(id string) (Content, error)
	// ReadDrawing reads a drawing for the given notebook and page ID.
	ReadDrawing(id, pageId string) (*Drawing, error)
}

func ReadNotebook(s Storage, id string) (*Notebook, error) {
	meta, err := s.ReadMetadata(id)
	if err != nil {
		return nil, err
	}

	content, err := s.ReadContent(id)
	if err != nil {
		return nil, err
	}

	pages := make([]*Page, len(content.Pages))
	for i, pageId := range content.Pages {
		pages[i] = &Page{NotebookID: id, ID: pageId}
	}

	// TODO: Read pagedata

	n := &Notebook{
		ID:      id,
		Meta:    meta,
		Content: content,
		Pages:   pages,
	}
	return n, nil
}

func ReadPage(s Storage, p *Page) error {
	d, err := s.ReadDrawing(p.NotebookID, p.ID)
	if err != nil {
		return err
	}
	p.Drawing = d

	// TODO: Read page meta

	return nil
}

func ReadFull(s Storage, id string) (*Notebook, error) {
	n, err := ReadNotebook(s, id)
	if err != nil {
		return nil, err
	}

	for _, p := range n.Pages {
		err = ReadPage(s, p)
		if err != nil {
			return nil, err
		}
	}

	return n, nil
}

type fsStorage struct {
	Base string
}

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
