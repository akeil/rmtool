package rm

// Notebook holds data for a complete notebook, including the drawings for all
// pages and metadata.
type Notebook struct {
	ID      string
	Meta    Metadata
	Content Content
	Pages   []*Page
}

// NewNotebook creates a new Notebook with the given ID.
func NewNotebook(id string) *Notebook {
	return &Notebook{ID: id}
}

type Page struct {
	NotebookID string
	ID         string
	Pagedata   Pagedata
	Meta       PageMetadata
	Drawing    *Drawing
}
