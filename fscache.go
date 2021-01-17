package rmtool

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/akeil/rmtool/internal/errors"
	"github.com/akeil/rmtool/internal/logging"
)

type fsCache struct {
	dir string
	mx  sync.RWMutex
}

// NewFilesystemCache returns a Cache implementation that stores cached data
// in the fiven directory.
func NewFilesystemCache(dir string) Cache {
	return &fsCache{dir: dir}
}

func (f *fsCache) Get(key string) (io.ReadCloser, error) {
	logging.Debug("Cache get %q", key)
	f.mx.RLock()
	defer f.mx.RUnlock()

	r, err := os.Open(f.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			logging.Debug("Cache miss %q", key)
			return nil, errors.NewNotFound("no cache entry for %q", key)
		}
		logging.Warning("Cache error %q", key)
		return nil, err
	}
	return r, nil
}

func (f *fsCache) Put(key string, r io.Reader) error {
	logging.Debug("Cache put %q", key)
	f.mx.Lock()
	defer f.mx.Unlock()

	err := f.mkdir()
	if err != nil {
		logging.Warning("Failed to create cahce directory %q: %v", f.dir, key)
		return err
	}

	w, err := os.Create(f.path(key))
	if err != nil {
		logging.Warning("Cache error %q", key)
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)

	return err
}

func (f *fsCache) Delete(key string) error {
	logging.Debug("Cache delete %q", key)
	f.mx.Lock()
	defer f.mx.Unlock()
	return os.Remove(f.path(key))
}

func (f *fsCache) path(key string) string {
	return filepath.Join(f.dir, key)
}

func (f *fsCache) mkdir() error {
	err := os.MkdirAll(f.dir, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}
