package fs

import (
	"io"
	"os"

	"github.com/akeil/rmtool/internal/logging"
)

// Move moves a file from src to dst.
// It tries os.Rename() first and falls back on "copy and delete".
//
// If src cannot be delted after a successful copy,
// NO error is returned and src remains as it was.
func Move(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Rename may have failed when moving across file systems
	// so try again w/ copy & delete.
	logging.Debug("Rename failed for %v -> %v, fall back on copy and delete", src, dst)
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	// A bit untidy, but we carry on even if we fail to clean up behind us.
	ignoredErr := os.Remove(src)
	if ignoredErr != nil {
		logging.Error("Failed to remove file %v", src)
	}

	return err
}
