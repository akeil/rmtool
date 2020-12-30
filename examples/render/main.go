package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/api"
	"akeil.net/akeil/rm/pkg/fs"
	"akeil.net/akeil/rm/pkg/render"
)

func main() {
	rm.SetLogLevel("debug")

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

	rc := render.NewContext("./data")

	var repo rm.Repository
	// filesystem
	repo = fs.NewRepository(dir)

	// api
	client, err := setup()
	if err != nil {
		log.Fatal(err)
	}
	repo = api.NewRepository(client, "/tmp/remarkable")

	root, err := rm.BuildTree(repo)
	if err != nil {
		log.Fatal(err)
	}
	root = root.Filtered(match)

	f := func(node *rm.Node) error {
		if !node.Leaf() {
			return nil
		}

		if node.Parent() == "trash" {
			return nil
		}

		doc, err := rm.ReadDocument(repo, node)
		if err != nil {
			log.Printf("Failed to read document %q", node.Name())
			return err
		}

		err = pngs(rc, doc)
		// err = pdf(rc, doc)
		if err != nil {
			log.Printf("Failed to render PDF for notebook %q", doc.ID())
		}
		return err
	}
	err = root.Walk(f)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("exit ok")
}

func pngs(rc *render.Context, doc *rm.Document) error {
	var wg sync.WaitGroup
	for i, p := range doc.Pages() {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()

			out := fmt.Sprintf("./out/drawing-%v.png", i)
			f, err := os.Create(out)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			w := bufio.NewWriter(f)
			err = rc.Page(doc, p, w)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("write %v", out)
			w.Flush()

		}(i, p)
	}

	wg.Wait()
	return nil
}

func pdf(rc *render.Context, n *rm.Document) error {
	// render to pdf
	p := filepath.Join("./out", n.Name()+".pdf")
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	return rc.PDF(n, w)
}

func setup() (*api.Client, error) {
	token, err := readToken()
	if err != nil {
		return nil, err
	}
	client := api.NewClient(api.StorageDiscoveryURL, api.NotificationsDiscoveryURL, api.AuthURL, token)

	return client, nil
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
