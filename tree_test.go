package rm

import (
	"testing"
)

func TestBuildTree(t *testing.T) {
	//s := NewFilesystemStorage("testdata")
	s := NewFilesystemStorage("/mnt/backup/remarkable/download/xochitl")
	_, err := BuildTree(s)
	if err != nil {
		t.Error(err)
	}
}
