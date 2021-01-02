package main

import (
	"fmt"

	"github.com/akeil/rmtool"
)

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

		fmt.Print(n.Name())
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
