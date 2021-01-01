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

	var err error
	switch command {
	case "ls":
		err = doLs(*format, *match, *pinned)
	case "get":
		err = doGet(*matchGet, *outDir, *mkDirs)
	case "put":
		err = doPut(*paths)
	default:
		err = fmt.Errorf("unknown command: %q", command)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func doLs(format, match string, pinned bool) error {
	repo, err := setupRepo()
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

func doGet(match, outDir string, mkDirs bool) error {
	repo, err := setupRepo()
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
	rc := render.NewContext("./data", p)

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

func doPut(paths []string) error {
	fmt.Println(paths)
	// split src(s) from dst
	var src []string
	var dst string
	if len(paths) == 0 {
		return fmt.Errorf("no source file(s) specified")
	} else if len(paths) == 1 {
		src = paths
	} else { // > 1
		src = paths[0 : len(paths)-1]
		dst = paths[len(paths)-1]
		if dst == "/" || dst == "." {
			dst = ""
		}
	}

	// expand srcs, Glob() will also filter any non-existing files
	srcs := make([]string, 0)
	for _, s := range src {
		fmt.Printf("expand %v\n", s)
		matches, err := filepath.Glob(s)
		if err != nil {
			return err
		}
		srcs = append(srcs, matches...)
	}
	if len(srcs) == 0 {
		return fmt.Errorf("no source file(s) specified")
	}

	for _, s := range srcs {
		fmt.Printf("Source: %v\n", s)
	}
	fmt.Printf("Destination: %v\n", dst)

	repo, err := setupRepo()
	if err != nil {
		return err
	}
	items, err := repo.List()
	if err != nil {
		return err
	}

	// TODO If a destination is specified, check if it is an existing
	// document or folder
	// if we have multiple source files,
	// the target must be an existing folder

	// TODO: if we have a single src, non-empty dst is either the target folder
	// OR the target name
	root := rm.BuildTree(items)
	if dst != "" {
		root = root.Filtered(rm.IsFolder, rm.MatchName(dst))
	}

	var group errgroup.Group
	for _, s := range srcs {
		srcPath := s // scope
		group.Go(func() error {
			return uploadPdf(repo, srcPath)
		})
	}

	return group.Wait()
}

func uploadPdf(repo rm.Repository, src string) error {
	_, file := filepath.Split(src)
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	_, ok := typesByExt[strings.ToLower(ext)]
	if !ok {
		return fmt.Errorf("unsupported file type %q", ext)
	}

	// TODO: single file upload uses dst as name
	doc, err := rm.NewPdf(base, func() (io.ReadCloser, error) {
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

// common ---------------------------------------------------------------------

func setupRepo() (rm.Repository, error) {
	client, err := setupClient()
	if err != nil {
		return nil, err
	}

	repo := api.NewRepository(client, "/tmp/remarkable")
	return repo, nil
}

func setupClient() (*api.Client, error) {
	var token string
	token, err := readToken()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	client := api.NewClient(api.StorageDiscoveryURL, api.NotificationsDiscoveryURL, api.AuthURL, token)

	if !client.IsRegistered() {
		err = register(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func register(client *api.Client) error {
	fmt.Printf("Register rmtool with remarkable\n")
	// TODO: prompt user
	code := ""
	token, err := client.Register(code)
	if err != nil {
		return err
	}

	saveToken(token)

	return nil
}

func readToken() (string, error) {
	tokenfile := "./data/device-token"
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

func saveToken(token string) {
	tokenfile := "./data/device-token"
	f, err := os.Create(tokenfile)
	if err != nil {
		fmt.Printf("Failed to save token to %q: %v\n", tokenfile, err)
	}
	defer f.Close()

	f.Write([]byte(token))
}
