package main

import (
	"fmt"
	"log"
	"os"

	"akeil.net/akeil/rm"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("wrong number of arguments")
		os.Exit(1)
	}
	dir := os.Args[1]

	s := rm.NewFilesystemStorage(dir)
	root, err := rm.BuildTree(s)
	if err != nil {
		log.Fatal(err)
	}

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
