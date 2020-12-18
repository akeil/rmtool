package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/render"
)

func main() {
	storage := rm.NewFilesystemStorage("testdata")
	id := "25e3a0ce-080a-4389-be2a-f6aa45ce0207"
	n, err := rm.ReadFull(storage, id)
	if err != nil {
		log.Fatal(err)
	}

	//pngs(storage, n)
	pdf(n)

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

func pdf(n *rm.Notebook) {
	// render to pdf
	f, err := os.Create("./out/notebook.pdf")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	render.RenderPDF(n, w)
}
