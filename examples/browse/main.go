package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"akeil.net/akeil/rm"
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

	s := rm.NewFilesystemStorage(dir)
	root, err := rm.BuildTree(s)
	if err != nil {
		log.Fatal(err)
	}

	root = root.Filtered(match)
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
