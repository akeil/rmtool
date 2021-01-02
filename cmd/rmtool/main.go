package main

import (
	"fmt"
	"image/color"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/api"
	"akeil.net/akeil/rm/pkg/render"
)

const (
	checkmark = "\u2713"
	crossmark = "\u2717"
	ellipsis  = "\u2026"
)

func main() {
	rm.SetLogLevel("warning")

	app := kingpin.New("rmtool", "reMarkable Tool")
	app.HelpFlag.Short('h')

	ls := app.Command("ls", "List notebooks").Default()
	var (
		pinned = ls.Flag("pinned", "Show only pinned items").Short('p').Bool()
		format = ls.Flag("format", "Output format").Short('f').Default("tree").String()
		match  = ls.Arg("match", "Name must match this").String()
	)

	get := app.Command("get", "Download one or more notebooks in PDF format")
	var (
		matchGet = get.Arg("match", "Name must match this").String()
		outDir   = get.Flag("output", "Output directory").Short('o').Default(".").String()
		mkDirs   = get.Flag("dirs", "Create subdirectories from tablet's folders").Short('d').Bool()
	)

	put := app.Command("put", "Upload PDF documents to reMArkable")
	var (
		paths = put.Arg("paths", "Source and destination paths").Strings()
	)

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	settings, err := loadSettings()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "ls":
		err = doLs(settings, *format, *match, *pinned)
	case "get":
		err = doGet(settings, *matchGet, *outDir, *mkDirs)
	case "put":
		err = doPut(settings, *paths)
	default:
		err = fmt.Errorf("unknown command: %q", command)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func doLs(s settings, format, match string, pinned bool) error {
	repo, err := setupRepo(s)
	if err != nil {
		return err
	}

	items, err := repo.List()
	if err != nil {
		return err
	}

	root := rm.BuildTree(items)
	filters := make([]rm.NodeFilter, 0)
	if match != "" {
		filters = append(filters, rm.IsDocument, rm.MatchName(match))
	}
	if pinned {
		filters = append(filters, rm.IsPinned)
	}

	root = root.Filtered(filters...)

	if len(root.Children) == 0 {
		fmt.Println("Found no matching notebooks.")
		return nil
	}

	root.Sort(rm.DefaultSort)

	fmt.Println("reMarkable Notebooks")
	fmt.Println("--------------------")

	switch format {
	case "tree":
		showTree(root, 0)
	case "list":
		showList(root)
	default:
		return fmt.Errorf("unsupported format, choose one of 'tree', 'list'")
	}

	return nil
}

func showList(n *rm.Node) {
	dateFormat := "Jan 02 2006, 15:04"

	show := func(n *rm.Node) error {
		if n.IsLeaf() {
			fmt.Print(" ")
		} else {
			fmt.Print("d")
		}

		if n.Pinned() {
			fmt.Print("*")
		} else {
			fmt.Print(" ")
		}

		fmt.Print(" ")
		fmt.Print(n.LastModified().Format(dateFormat))
		fmt.Print(" | ")
		fmt.Print(n.Name())
		fmt.Println()

		return nil
	}
	n.Walk(show)
}

func showTree(n *rm.Node, level int) {
	if level > 0 {
		for i := 1; i < level; i++ {
			fmt.Print("  ")
		}

		if n.IsLeaf() {
			fmt.Print("- ")
		} else {
			fmt.Print("+ ")
		}

		fmt.Printf(n.Name())
		if n.Pinned() {
			fmt.Print(" *")
		}

		fmt.Println()
	}

	if !n.IsLeaf() {
		for _, c := range n.Children {
			showTree(c, level+1)
		}
	}
}

// get ------------------------------------------------------------------------

func doGet(s settings, match, outDir string, mkDirs bool) error {
	repo, err := setupRepo(s)
	if err != nil {
		return err
	}

	items, err := repo.List()
	if err != nil {
		return err
	}

	root := rm.BuildTree(items)
	root = root.Filtered(rm.IsDocument, rm.MatchName(match))

	if len(root.Children) == 0 {
		fmt.Printf("No matching documents for %q\n", match)
		return nil
	}

	brushes := map[rm.BrushColor]color.Color{
		rm.Black: color.RGBA{0, 20, 120, 255},   // dark blue
		rm.Gray:  color.RGBA{35, 110, 160, 255}, // light/gray blue
		rm.White: color.White,
	}
	p := render.NewPalette(color.White, brushes)
	rc := render.NewContext(s.dataDir, p)

	var group errgroup.Group
	root.Walk(func(n *rm.Node) error {
		if n.Type() == rm.CollectionType {
			return nil
		}
		group.Go(func() error {
			return renderPdf(rc, repo, n, outDir, mkDirs)
		})
		return nil
	})
	return group.Wait()
}

func renderPdf(rc *render.Context, repo rm.Repository, item *rm.Node, outDir string, mkDirs bool) error {
	fmt.Printf("%v download %q\n", ellipsis, item.Name())
	doc, err := rm.ReadDocument(repo, item)
	if err != nil {
		fmt.Printf("%v Failed to download %q: %v\n", crossmark, item.Name(), err)
		return err
	}

	// Mirror the directory structure from the tablet
	p := item.Path()
	p = p[1:] // drop root element
	if mkDirs && len(p) != 0 {
		outDir = filepath.Join(outDir, filepath.Join(p...))
		err = os.MkdirAll(outDir, 0755)
		if err != nil {
			fmt.Printf("%v Failed to create directory %q: %v\n", crossmark, outDir, err)
			return err
		}
	}

	path := filepath.Join(outDir, doc.Name()+".pdf")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Printf("%v render %q\n", ellipsis, item.Name())
	err = rc.Pdf(doc, f)

	if err != nil {
		fmt.Printf("%v Failed to render %q: %v\n", crossmark, item.Name(), err)
		return err
	}

	fmt.Printf("%v document %q saved as %q.\n", checkmark, item.Name(), path)
	return nil
}

// put ------------------------------------------------------------------------

var typesByExt = map[string]rm.FileType{
	rm.Pdf.Ext(): rm.Pdf,
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
	root := rm.BuildTree(items)

	var dstNode *rm.Node
	var dstName string
	var dstType rm.NotebookType
	if dst == "" {
		dstNode = root
		dstName = ""
		dstType = rm.CollectionType
	} else {
		dstNode, dstName = determineUploadDst(root, dst)
		if dstNode != nil {
			dstType = dstNode.Type()
		}
	}

	if dstNode == nil {
		return fmt.Errorf("Destination path %q does not exist", dst)
	}

	// not all combinations are allowed
	if len(src) == 1 {
		if dstType == rm.DocumentType {
			// replace existing document
			// TODO implement
			return fmt.Errorf("replace existing document is not implemented")
		}
		// upload to dstNode
		// nmae = dstName or from filename
	} else { // multiple source files
		if dstType == rm.DocumentType || dstName != "" {
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
func uploadPdf(repo rm.Repository, src string, dstName string, dstNode *rm.Node) error {
	if dstName == "" {
		_, file := filepath.Split(src)
		ext := filepath.Ext(file)
		dstName = strings.TrimSuffix(file, ext)

	}

	doc, err := rm.NewPdf(dstName, dstNode.ID(), func() (io.ReadCloser, error) {
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
func determineUploadDst(root *rm.Node, path string) (*rm.Node, string) {
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
			if strings.ToLower(name) == strings.ToLower(child.Name()) {
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
		if node.Type() == rm.CollectionType {
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

// common ---------------------------------------------------------------------

type settings struct {
	dataDir  string
	cacheDir string
}

func loadSettings() (settings, error) {
	var s settings
	// TODO: from env vars
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		// TODO linux only
		home, err := os.UserHomeDir()
		if err != nil {
			return s, err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	s.dataDir = filepath.Join(dataHome, "rmtool")

	cacheHome, err := os.UserCacheDir()
	if err != nil {
		return s, err
	}
	s.cacheDir = filepath.Join(cacheHome, "rmtool")

	return s, nil
}

func setupRepo(s settings) (rm.Repository, error) {
	client, err := setupClient(s)
	if err != nil {
		return nil, err
	}

	repo := api.NewRepository(client, s.cacheDir)
	return repo, nil
}

func setupClient(s settings) (*api.Client, error) {
	var token string
	token, err := readToken(s)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	client := api.NewClient(api.StorageDiscoveryURL, api.NotificationsDiscoveryURL, api.AuthURL, token)

	if !client.IsRegistered() {
		err = register(s, client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func register(s settings, client *api.Client) error {
	fmt.Printf("Register rmtool with remarkable\n")
	// TODO: prompt user
	code := ""
	token, err := client.Register(code)
	if err != nil {
		return err
	}

	saveToken(s, token)

	return nil
}

func readToken(s settings) (string, error) {
	tokenfile := filepath.Join(s.dataDir, "device-token")
	f, err := os.Open(tokenfile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	d, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(d), err
}

func saveToken(s settings, token string) {
	tokenfile := filepath.Join(s.dataDir, "device-token")
	f, err := os.Create(tokenfile)
	if err != nil {
		fmt.Printf("Failed to save token to %q: %v\n", tokenfile, err)
	}
	defer f.Close()

	f.Write([]byte(token))
}
