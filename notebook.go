package rmtool

import (
	"fmt"
	"os"
	"path/filepath"
)

type Notebook struct {
	Base    string
	ID      string
	Meta    Metadata
	Content Content
	Pages   []*Page
}

func NewNotebook(baseDir, id string) *Notebook {
	return &Notebook{Base: baseDir, ID: id}
}

func (n *Notebook) Read() error {
	mp := filepath.Join(n.Base, n.ID+".metadata")
	m, err := ReadMetadata(mp)
	if err != nil {
		return err
	}
	n.Meta = m

	cp := filepath.Join(n.Base, n.ID+".content")
	c, err := ReadContent(cp)
	if err != nil {
		return err
	}
	n.Content = c

	n.Pages = make([]*Page, len(n.Content.Pages))
	for i, pageID := range n.Content.Pages {
		p := NewPage(n.Base, n.ID, pageID)
		n.Pages[i] = p
	}

	return nil
}

type Page struct {
	Base    string
	ID      string
	Meta    PageMetadata
	Drawing *Drawing
}

func NewPage(baseDir, notebookID, pageID string) *Page {
	return &Page{
		Base: filepath.Join(baseDir, notebookID),
		ID:   pageID,
	}
}

func (p *Page) ReadDrawing() error {
	src := filepath.Join(p.Base, p.ID+".rm")
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("cannot read rm file %q. Error: %v", src, err)
	}
	defer r.Close()

	d, err := ReadDrawing(r)
	if err != nil {
		return fmt.Errorf("cannot read rm file %q. Error: %v", src, err)
	}

	p.Drawing = d
	return nil
}
