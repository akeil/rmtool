package rm

import (
	"fmt"
)

// Storage is the interface for a storage backend that holds notebooks.
type Storage interface {
	// ReadMetadata reads the metadata for a notebook with the given ID.
	ReadMetadata(id string) (Metadata, error)
	// ReadContent reads the content for a notebook.
	ReadContent(id string) (Content, error)
	// ReadDrawing reads a drawing for the given notebook and page ID.
	ReadDrawing(id, pageId string) (*Drawing, error)
	// ReadPagedata reads pagedata (templates) for a notebook id
	ReadPagedata(id string) ([]Pagedata, error)
}

// ReadNotebook reads a notebook with all metadata from the given storage.
// It initializes the pages but does not read the individual page data.
func ReadNotebook(s Storage, id string) (*Notebook, error) {
	meta, err := s.ReadMetadata(id)
	if err != nil {
		return nil, err
	}

	content, err := s.ReadContent(id)
	if err != nil {
		return nil, err
	}

	pagedata, err := s.ReadPagedata(id)
	if err != nil {
		return nil, err
	}

	if len(content.Pages) != len(pagedata) {
		return nil, fmt.Errorf("inconsistent data: %v pages vs. %v pagedata entries", len(content.Pages), len(pagedata))
	}

	pages := make([]*Page, len(content.Pages))
	for i, pageId := range content.Pages {
		pages[i] = &Page{
			NotebookID: id,
			ID:         pageId,
			Pagedata:   pagedata[i],
		}
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

// ReadPage reads the data for a single page from storage.
// This includes the drawing and the metadata.
// It sets the Drawing and MEta fields of the given page.
func ReadPage(s Storage, p *Page) error {
	d, err := s.ReadDrawing(p.NotebookID, p.ID)
	if err != nil {
		return err
	}
	p.Drawing = d

	// TODO: Read page meta

	return nil
}

// ReadFull reads a notebook and all its pages from storage.
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
