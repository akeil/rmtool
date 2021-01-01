package main

import (
	"fmt"
	"log"
	"os"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/fs"
)

func main() {
	rm.SetLogLevel("debug")

	var dir string
	match := make([]rm.NodeFilter, 0)
	if len(os.Args) == 2 {
		dir = os.Args[1]
	} else if len(os.Args) == 3 {
		dir = os.Args[1]
		match = append(match, rm.MatchName(os.Args[2]), rm.IsDocument)
	} else {
		fmt.Println("wrong number of arguments")
		os.Exit(1)
	}

	repo := fs.NewRepository(dir)
	root, err := rm.BuildTree(repo)
	if err != nil {
		log.Fatal(err)
	}

	root = root.Filtered(match...)
	root.Sort(rm.DefaultSort)

	for _, c := range root.Children {
		show(c, 0)
	}

	os.Exit(0)
}

func show(n *rm.Node, level int) {
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}

	if n.Leaf() {
		fmt.Print("- ")
	} else {
		fmt.Print("+ ")
	}
	fmt.Print(n.Name())

	if n.Pinned() {
		fmt.Print(" *")
	}
	fmt.Println()

	if !n.Leaf() {
		for _, c := range n.Children {
			show(c, level+1)
		}
	}
}
