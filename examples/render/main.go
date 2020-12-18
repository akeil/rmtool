package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/render"
)

func main() {
	var dir string
	var match rm.NodeFilter
	if len(os.Args) == 2 {
		dir = os.Args[1]
		match = func(n *rm.Node) bool {
			return true
		}
	} else if len(os.Args) == 3 {
		dir = os.Args[1]
		s := strings.ToLower(os.Args[2])
		match = func(n *rm.Node) bool {
			return strings.Contains(strings.ToLower(n.Name()), s)
		}
	} else {
		fmt.Println("wrong number of arguments")
		os.Exit(1)
	}

	storage := rm.NewFilesystemStorage(dir)
	root, err := rm.BuildTree(storage)
	if err != nil {
		log.Fatal(err)
	}
	root = root.Filtered(match)

	f := func(node *rm.Node) error {
		if !node.Leaf() {
			return nil
		}

		if node.Parent.ID == "trash" {
			return nil
		}

		n, err := rm.ReadFull(storage, node.ID)
		if err != nil {
			log.Printf("Failed to read notebook %q", node.Name())
			return err
		}

		//pngs(storage, n)
		err = pdf(n)
		if err != nil {
			log.Printf("Failed to render PDF for notebook %q", n.Meta.VisibleName)
		}
		return err
	}
	err = root.Walk(f)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("exit ok")
}

func pngs(storage rm.Storage, n *rm.Notebook) {
	var wg sync.WaitGroup
	for i, p := range n.Pages {
		wg.Add(1)
		go func(i int, p *rm.Page) {
			defer wg.Done()
			//log.Printf("Read page %v", i)
			//err := rm.ReadPage(storage, p)
			//if err != nil {
			//	log.Fatal(err)
			//}

			err := p.Drawing.Validate()
			if err != nil {
				log.Printf("Found validation error: %v", err)
			}

			out := fmt.Sprintf("./out/drawing-%v.png", i)
			f, err := os.Create(out)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			w := bufio.NewWriter(f)
			err = render.RenderPage(p, w)
			if err != nil {
				log.Fatal(err)
			}
			w.Flush()

		}(i, p)
	}

	wg.Wait()
}

func pdf(n *rm.Notebook) error {
	// render to pdf
	p := filepath.Join("./out", n.Meta.VisibleName+".pdf")
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	return render.RenderPDF(n, w)
}
