package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/akeil/rm"
	"github.com/akeil/rm/pkg/lines"
	"github.com/akeil/rm/pkg/render"
)

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

	brushes := map[lines.BrushColor]color.Color{
		lines.Black: color.RGBA{0, 20, 120, 255},   // dark blue
		lines.Gray:  color.RGBA{35, 110, 160, 255}, // light/gray blue
		lines.White: color.White,
	}
	yellow := color.RGBA{240, 240, 80, 255}
	p := render.NewPalette(color.White, yellow, brushes)
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
