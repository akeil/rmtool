package fs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akeil/rmtool"
	"github.com/akeil/rmtool/internal/errors"
	fsx "github.com/akeil/rmtool/internal/fs"
	"github.com/akeil/rmtool/internal/logging"
)

type repo struct {
	base string
}

// NewRepository creates a repository backed by the local file system.
//
// The given path should point to a directory similar to the storage directory
// on the remarkable tablet.
func NewRepository(path string) rmtool.Repository {
	return &repo{
		base: path,
	}
}

func (r *repo) List() ([]rmtool.Meta, error) {
	logging.Debug("List files from %q", r.base)

	files, err := os.ReadDir(r.base)
	if err != nil {
		return nil, err
	}

	l := make([]rmtool.Meta, 0)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".metadata" {
			id := strings.TrimSuffix(f.Name(), ".metadata")
			m, err := r.readItem(id)
			if err != nil {
				return nil, err
			}
			l = append(l, m)
		}
	}

	return l, err
}

func (r *repo) readItem(id string) (rmtool.Meta, error) {
	p := filepath.Join(r.base, id+".metadata")
	meta, err := readMetadata(p)
	if err != nil {
		return nil, err
	}
	return metaWrapper{id: id, i: &meta, repo: r}, nil
}

func (r *repo) Update(m rmtool.Meta) error {
	logging.Debug("Update entry with id %q, version %v", m.ID(), m.Version())
	err := m.Validate()
	if err != nil {
		return err
	}
	err = r.checkParent(m.Parent())
	if err != nil {
		return err
	}

	p := filepath.Join(r.base, m.ID()+".metadata")
	o, err := readMetadata(p)
	if err != nil {
		return err
	}

	// check the version
	if m.Version() != o.Version {
		return fmt.Errorf("version mismatch %d != %d", m.Version(), o.Version)
	}

	o.Version++
	o.LastModified = Timestamp{time.Now()}

	// assumption: we need to set these if we write to the tablet
	o.Synced = false
	o.MetadataModified = true

	// apply the changes
	o.VisibleName = m.Name()
	o.Pinned = m.Pinned()
	o.Parent = m.Parent()
	o.Type = m.Type()

	// to tempfile
	f, err := os.CreateTemp("", "rm-*.json")
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

	return fsx.Move(f.Name(), p)
}

func (r *repo) Upload(d *rmtool.Document) error {
	err := d.Validate()
	if err != nil {
		return err
	}
	err = r.checkParent(d.Parent())
	if err != nil {
		return err
	}

	// We will write everything to a temporary directory,
	// then move to the target dir
	tmp, err := os.MkdirTemp("", "rm-upload-*")
	if err != nil {
		return err
	}

	// Cleanup:
	// on success, this will remove the empty temp dir,
	// on error, this will remove the files written so far.
	defer func() {
		logging.Debug("Cleanup %q", tmp)
		cleanupErr := os.RemoveAll(tmp)
		if cleanupErr != nil {
			logging.Warning("Error during cleanup: %v", cleanupErr)
		}
	}()

	logging.Debug("Write individual files to temp dir %q...", tmp)

	// Capture all the files we have created.
	files := make(map[string]string)

	// Set up a factory function to create writers for tempfiles.
	w := func(path ...string) (io.WriteCloser, error) {
		if len(path) == 0 {
			return nil, fmt.Errorf("path must not be empty")
		}

		parts := []string{tmp}
		parts = append(parts, path...)

		// Do we need to create a subdirectory?
		if len(path) > 1 {
			subDir := filepath.Join(parts[0 : len(parts)-1]...)
			err = os.Mkdir(subDir, 0755)
			if err != nil {
				if !os.IsExist(err) {
					return nil, err
				}
			}
		}

		abs := filepath.Join(parts...)
		rel := filepath.Join(path...)

		logging.Debug("Create %q", abs)
		f, e := os.Create(abs)
		if e != nil {
			return nil, e
		}

		// Capture the file we are going to write.
		files[rel] = abs

		return f, nil
	}

	// Write the metadata entry.
	logging.Debug("Write metadata")
	meta := Metadata{
		LastModified:     Timestamp{time.Now()},
		Version:          d.Version(),
		Parent:           d.Parent(),
		Pinned:           d.Pinned(),
		Type:             d.Type(),
		VisibleName:      d.Name(),
		LastOpenedPage:   0,
		Deleted:          false,
		MetadataModified: false,
		Modified:         false,
		Synced:           false,
	}

	mw, err := w(fmt.Sprintf("%v.metadata", d.ID()))
	if err != nil {
		return err
	}
	defer mw.Close()
	err = json.NewEncoder(mw).Encode(meta)
	if err != nil {
		return err
	}

	// Let the document write individual parts.
	logging.Debug("Write document parts...")

	err = d.Write(r, w)
	if err != nil {
		return err
	}

	// TODO: if we have an error during one of the moves,
	// the partially transferred content in dst needs cleanup

	// We always create the <ID>/ subdirectory, even if it will be empty.
	// At least, this seems to be the behaviour of the remarkable tablet.
	pagesDir := filepath.Join(r.base, d.ID())
	err = os.Mkdir(pagesDir, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	// Move everything to the target directory.
	logging.Debug("Move files to %q...", r.base)
	for rel, src := range files {
		dst := filepath.Join(r.base, rel)
		// Create a subdirectory if needed.
		dir, _ := filepath.Split(rel)
		if dir != "" {
			logging.Debug("Create subdirectory %q", dir)
			absDir := filepath.Join(r.base, dir)
			err := os.Mkdir(absDir, 0755)
			if err != nil {
				if !os.IsExist(err) {
					return err
				}
			}
		}
		logging.Debug("Move %v", rel)

		err = fsx.Move(src, dst)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r repo) PagePrefix(id string, index int) string {
	return id
}

func (r *repo) Reader(id string, version uint, path ...string) (io.ReadCloser, error) {
	parts := []string{r.base}
	parts = append(parts, path...)
	p := filepath.Join(parts...)

	logging.Debug("Create reader for %q\n", p)

	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return f, errors.NewNotFound(err.Error())
	}
	return f, err
}

func (r *repo) checkParent(parentID string) error {
	if parentID == "" {
		return nil
	}

	parent, err := r.readItem(parentID)
	if err != nil {
		return err
	}

	if parent.Type() != rmtool.CollectionType {
		return fmt.Errorf("parent with id %q is no a collection (type=%v)", parentID, parent.Type())
	}

	return nil
}

func readMetadata(path string) (Metadata, error) {
	var m Metadata
	r, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, errors.NewNotFound("no metadata file at %q", path)
		}
		return m, errors.Wrap(err, "failed to read metadata for %q", path)
	}
	defer r.Close()

	err = json.NewDecoder(r).Decode(&m)
	if err != nil {
		return m, errors.Wrap(err, "failed to read metadata for %q", path)
	}

	return m, err
}

type metaWrapper struct {
	id   string
	i    *Metadata
	repo *repo
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

func (m metaWrapper) Type() rmtool.NotebookType {
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

func (m metaWrapper) Validate() error {
	return m.i.Validate()
}
