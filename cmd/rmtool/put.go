package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/akeil/rmtool"
)

var typesByExt = map[string]rmtool.FileType{
	rmtool.Pdf.Ext(): rmtool.Pdf,
}

func doPut(s settings, paths []string) error {
	src, dst := normalizeSrcDst(paths)

	if len(src) == 0 {
		return fmt.Errorf("no source file(s) specified")
	}

	err := checkSrcFormat(src)
	if err != nil {
		return err
	}

	repo, err := setupRepo(s)
	if err != nil {
		return err
	}
	items, err := repo.List()
	if err != nil {
		return err
	}
	root := rmtool.BuildTree(items)

	var dstNode *rmtool.Node
	var dstName string
	var dstType rmtool.NotebookType
	if dst == "" {
		dstNode = root
		dstName = ""
		dstType = rmtool.CollectionType
	} else {
		dstNode, dstName = determineUploadDst(root, dst)
		if dstNode != nil {
			dstType = dstNode.Type()
		}
	}

	if dstNode == nil {
		return fmt.Errorf("destination path %q does not exist", dst)
	}

	// not all combinations are allowed
	if len(src) == 1 {
		if dstType == rmtool.DocumentType {
			// replace existing document
			// TODO implement
			return fmt.Errorf("replace existing document is not implemented")
		}
		// upload to dstNode
		// nmae = dstName or from filename
	} else { // multiple source files
		if dstType == rmtool.DocumentType || dstName != "" {
			return fmt.Errorf("cannot upload multiple documents to a single target document")
		}
		// upload to dstNode,
		// name = from filename
	}

	// TODO when name is chosen from filename, it may still refer to an existing name
	// currently, this will lead to duplicate names in the same folder
	// technically OK, but not what we want

	var group errgroup.Group
	for _, s := range src {
		srcPath := s // scope
		group.Go(func() error {
			return uploadPdf(repo, srcPath, dstName, dstNode)
		})
	}

	return group.Wait()
}

// upload a single pdf
func uploadPdf(repo rmtool.Repository, src string, dstName string, dstNode *rmtool.Node) error {
	if dstName == "" {
		_, file := filepath.Split(src)
		ext := filepath.Ext(file)
		dstName = strings.TrimSuffix(file, ext)

	}

	doc, err := rmtool.NewPdf(dstName, dstNode.ID(), func() (io.ReadCloser, error) {
		f, err := os.Open(src)
		if err != nil {
			return nil, err
		}
		return f, nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("%v upload %q\n", ellipsis, doc.Name())
	err = repo.Upload(doc)
	if err != nil {
		return err
	}

	fmt.Printf("%v %q uploaded\n", checkmark, doc.Name())
	return nil
}

// determine the upload destination from a given destination path.
//
// If the path matches a node exactly, that node is returned
// (the node can refer to a folder or a document).
//
// If the path matches except the last component,
// the mtching node is returned IF it is a folder
// and the last path component is returned as a string.
func determineUploadDst(root *rmtool.Node, path string) (*rmtool.Node, string) {
	// normalizes path:
	// /foo/bar  =>  foo/bar
	// foo/bar/  =>  foo/bar
	// foo//bar  =>  foo/bar
	norm := make([]string, 0)
	parts := strings.Split(path, "/")
	for _, s := range parts {
		if s != "" {
			norm = append(norm, s)
		}
	}

	// walk the tree, accepting the FIRST child that matches a path component
	// stop on first non-match
	var consumed int
	var found bool
	node := root
	for _, name := range norm {
		found = false
		for _, child := range node.Children {
			if strings.EqualFold(name, child.Name()) {
				found = true
				node = child
			}
		}
		if !found {
			break
		}
		consumed++
	}
	// path components we have NOT matched
	unmatched := norm[consumed:]

	if len(unmatched) == 0 {
		// exact match
		return node, ""
	} else if len(unmatched) == 1 {
		if node.Type() == rmtool.CollectionType {
			// matched "new document in parent folder"
			return node, unmatched[0]
		}
	}

	// no match
	return nil, ""
}

// Split a list of paths into a list of SRC's and a single DST.
// If the initial list contains less than two entries, DST is empty,
// otherwise, DST is the last element from the list.
func normalizeSrcDst(paths []string) ([]string, string) {
	src := make([]string, 0)
	var dst string

	var temp []string
	if len(paths) == 0 {
		return src, dst // empty
	} else if len(paths) == 1 {
		temp = paths
	} else { // > 1
		temp = paths[0 : len(paths)-1]
		dst = paths[len(paths)-1]
		if dst == "/" || dst == "." {
			dst = ""
		}
	}

	// expand SRCs
	// Glob() will also filter any non-existing files
	// TODO: this might be redundant
	for _, s := range temp {
		fmt.Printf("expand %v\n", s)
		matches, err := filepath.Glob(s)
		if err != nil {
			fmt.Println(err)
			continue
		}
		src = append(src, matches...)
	}

	return src, dst
}

// Check if all of the given src paths are supported file types.
func checkSrcFormat(src []string) error {
	for _, s := range src {
		_, file := filepath.Split(s)
		ext := filepath.Ext(file)
		_, ok := typesByExt[strings.ToLower(ext)]
		if !ok {
			return fmt.Errorf("unsupported file type %q", ext)
		}
	}
	return nil
}
