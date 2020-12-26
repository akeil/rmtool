package api

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/logging"
)

type repo struct {
	client  *Client
	dataDir string
	mx      sync.RWMutex
}

// NewRepository creates a Repository with the reMarkable cloud service as
// backend.
//
// The supplied dataDir is used to cache downloaded content.
func NewRepository(c *Client, dataDir string) rm.Repository {
	return &repo{
		client:  c,
		dataDir: dataDir,
	}
}

func (r *repo) List() ([]rm.Meta, error) {
	logging.Debug("Repository.List")
	items, err := r.client.List()
	if err != nil {
		return nil, err
	}

	rv := make([]rm.Meta, len(items))
	for i, item := range items {
		rv[i] = metaWrapper{i: item, r: r}
	}

	return rv, nil
}

func (r *repo) Update(m rm.Meta) error {
	item := Item{
		ID:          m.ID(),
		Version:     int(m.Version()),
		Type:        m.Type(),
		VisibleName: m.Name(),
		Bookmarked:  m.Pinned(),
		Parent:      m.Parent(),
	}
	return r.client.update(item)
}

func (r *repo) reader(id string, version uint, path ...string) (io.ReadCloser, error) {

	// Attempt to read from cache, download if not exists or corrupt
	p := r.cachePath(id, version)

	r.mx.RLock()
	defer r.mx.RUnlock()
	zr, err := zip.OpenReader(p)

	// If the file does not exist or is otherwise unusable,
	// download new and try again.
	if err != nil {
		// CAREFUL: we need a write lock when we downloading,
		// so we release our read lock for a moment.
		//
		// This relies on the RLock being acquired before
		// AND relies on the (defer) RUnlock being called before.
		r.mx.RUnlock()
		r.downloadToCache(id, version)
		r.mx.RLock()

		zr, err = zip.OpenReader(p)
		if err != nil {
			return nil, err
		}
	}

	// Read the desired entry from the zip file
	match := strings.Join(path, "/")
	var entry *zip.File
	for _, zf := range zr.File {
		if zf.Name == match {
			entry = zf
			break
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("no zip entry found with name %q", match)
	}

	// return a reader for the file entry
	// closing the reader should close the zip reader
	return entry.Open()
}

func (r *repo) downloadToCache(id string, version uint) error {
	// Retreive the BlobURLGet
	i, err := r.client.fetchItem(id)
	if err != nil {
		return err
	}

	// Download to temp
	f, err := ioutil.TempFile("", "rm_"+id+"_*.zip")
	if err != nil {
		return err
	}
	defer f.Close()
	// cleanup: delete the tempfile (errors ignored)
	defer func() {
		_, err := os.Stat(f.Name())
		if err != nil {
			if os.IsNotExist(err) {
				return
			}
			logging.Warning("Unexpected error: %v\n", err)
			return
		}
		err = os.Remove(f.Name())
		if err != nil {
			logging.Warning("Unexpected error: %v\n", err)
		}
	}()

	logging.Debug("Download blob to %q\n", f.Name())
	err = r.client.fetchBlob(i.BlobURLGet, f)
	if err != nil {
		return err
	}

	// Lock for writing.
	r.mx.Lock()
	defer r.mx.Unlock()

	// Prepare the destination directory
	err = os.MkdirAll(r.dataDir, 0755)
	if err != nil {
		return fmt.Errorf("could not create cache dir: %v", err)
	}

	// Move to destination dir
	p := r.cachePath(id, version)
	logging.Debug("Move archive blob to %q\n", p)
	err = os.Rename(f.Name(), p)
	if err != nil {
		return err
	}

	// This should presumably remove the one outdated entry (if any).
	go r.cleanCache()

	return nil
}

func (r *repo) cachePath(id string, version uint) string {
	return filepath.Join(r.dataDir, fmt.Sprintf("%v_%v.zip", id, version))
}

// cleanCache removes outdated versions from the cache.
func (r *repo) cleanCache() {
	// Filenames look like this:
	//
	//   <ID>_<Version>.zip
	//
	// We want to keep only the highest version for each ID.

	// Hold the write lock the whole time as we read and change the directory..
	r.mx.Lock()
	defer r.mx.Unlock()

	// List all cached files.
	files, err := ioutil.ReadDir(r.dataDir)
	if err != nil {
		logging.Warning("Could not list cache directory: %v", err)
		return
	}

	// Determine which versions we have for each id.
	versions := make(map[string][]int)
	for _, f := range files {
		base := filepath.Base(f.Name())
		parts := strings.Split(base, "_")
		if len(parts) != 2 {
			logging.Warning("Clean cache: encountered unexpected filename %q", base)
			continue
		}
		id := parts[0]
		// "123.zip" => 123
		v, err := strconv.Atoi(strings.TrimSuffix(parts[1], ".zip"))
		if err != nil {
			logging.Warning("Clean cache: error retrieving version from filename %q, %v", base, err)
			continue
		}

		if versions[id] == nil {
			versions[id] = make([]int, 0)
		}
		versions[id] = append(versions[id], v)
	}

	// Delete all versions except the highest
	for id, v := range versions {
		if len(v) < 2 {
			continue
		}
		sort.Ints(v)
		for i := 0; i < len(v)-1; i++ {
			p := r.cachePath(id, uint(v[i]))
			logging.Info("Remove outdated version from cache: %q", p)
			err = os.Remove(p)
			if err != nil {
				logging.Warning("Unexpected error removing old cache entry: %v", err)
				continue
			}
		}
	}
}

// implement the Meta interface for an Item
type metaWrapper struct {
	i Item
	r *repo
}

func (m metaWrapper) Reader(path ...string) (io.ReadCloser, error) {
	return m.r.reader(m.ID(), m.Version(), path...)
}

func (m metaWrapper) ID() string {
	return m.i.ID
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
	return m.i.Bookmarked
}

func (m metaWrapper) SetPinned(b bool) {
	m.i.Bookmarked = b
}

func (m metaWrapper) LastModified() time.Time {
	return m.i.ModifiedClient.Time
}

func (m metaWrapper) Parent() string {
	return m.i.Parent
}

func (m metaWrapper) PagePrefix(id string, index int) string {
	return fmt.Sprintf("%d", index)
}
