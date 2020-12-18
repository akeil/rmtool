package rm

import (
	"fmt"
	"sort"
	"strings"
)

// Node is the representation for an entry in the content tree.
// A not can either be a document or a collection (which has child nodes).
type Node struct {
	ID       string
	Parent   *Node
	Children []*Node
	meta     Metadata
}

func newNode(id string, m Metadata) *Node {
	return &Node{
		ID:       id,
		Children: make([]*Node, 0),
		meta:     m,
	}
}

func (n *Node) Type() NotebookType {
	return n.meta.Type
}

func (n *Node) Name() string {
	return n.meta.VisibleName
}

func (n *Node) Root() bool {
	return n.ID == ""
}

func (n *Node) Leaf() bool {
	return n.Type() != CollectionType && !n.Root()
}

func (n *Node) Pinned() bool {
	return n.meta.Pinned
}

// Walk applies the given function to the subtree starting at this node,
// (including this node). Returns the first error that is encountered or nil.
func (n *Node) Walk(f func(n *Node) error) error {
	err := f(n)
	if err != nil {
		return err
	}

	for _, c := range n.Children {
		err = c.Walk(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// Sort sorts the subtree starting at this node by the given sort rule.
// Sorting is in-place.
func (n *Node) Sort(compare func(*Node, *Node) bool) {
	f := func(i, j int) bool {
		one := n.Children[i]
		other := n.Children[j]
		return compare(one, other)
	}
	sort.Slice(n.Children, f)

	for _, c := range n.Children {
		c.Sort(compare)
	}
}

// addChild adds a child node to this node and sets the Parent field
// of the child.
func (n *Node) addChild(child *Node) {
	n.Children = append(n.Children, child)
	child.Parent = n
}

// Put attempts to accomodate this node in the subtree starting at this node.
// The child node can be added as an immediate child or grandchild.
// Returns `true` if the node could be added to the tree.
func (n *Node) put(other *Node) bool {
	if other.meta.Parent == n.ID {
		n.addChild(other)
		return true
	}

	for _, c := range n.Children {
		if c.put(other) {
			return true
		}
	}

	return false
}

// BuildTree creates a tree view of all items in the given storage.
// Returns the root node.
func BuildTree(s Storage) (*Node, error) {
	l, err := s.List()
	if err != nil {
		return nil, err
	}

	root := &Node{}
	root.addChild(newNode("trash", Metadata{
		Type:        CollectionType,
		VisibleName: "Trash",
	}))

	var m Metadata
	nodes := make([]*Node, 0)
	for _, id := range l {
		m, err = s.ReadMetadata(id)
		if err != nil {
			return nil, err
		}
		if !m.Deleted {
			nodes = append(nodes, newNode(id, m))
		}
	}

	change := false
	for {
		change = false
		remaining := make([]*Node, 0)
		for _, n := range nodes {
			if root.put(n) {
				change = true
			} else {
				remaining = append(remaining, n)
			}
		}
		nodes = remaining
		if change == false {
			break
		}
	}

	if len(nodes) != 0 {
		return nil, fmt.Errorf("could not put all notes into the tree")
	}

	return root, nil
}

// DefaultSort is the comparsion function to sort nodes in the content tree
// with folders before documents and by name (case-insensitive).
// Pinned notes come before unpinned ones within a folder.
// The "Trash" folder comes last.
func DefaultSort(one, other *Node) bool {
	// tell if  one <  other
	// special case - Trash goes last
	if one.ID == "trash" {
		return false
	} else if other.ID == "trash" {
		return true
	}

	// collections before content
	if one.Leaf() && !other.Leaf() {
		return false
	} else if other.Leaf() && !one.Leaf() {
		return true
	}

	// pinned before unpinned
	if one.Pinned() && !other.Pinned() {
		return true
	} else if other.Pinned() && !one.Pinned() {
		return false
	}

	// special case, equal display names, fall back on ID
	if one.Name() == other.Name() {
		return one.ID < other.ID
	}

	// by name, case-insensitive
	return strings.ToLower(one.Name()) < strings.ToLower(other.Name())

}

// A NodeFilter is a function that can be used to test whether a node should
// be included in a filtered subset or not.
type NodeFilter func(n *Node) bool

// Filtered returns a new node that is the root of a subtree starting at this node.
// The subtree cill contain only nodes that match the given NodeFilter
// and the parent folders of the matched nodes.
func (n *Node) Filtered(accept NodeFilter) *Node {
	root := newNode(n.ID, n.meta)
	for _, c := range n.Children {
		if c.Leaf() {
			if accept(c) {
				root.addChild(newNode(c.ID, c.meta))
			}
		} else {
			x := c.Filtered(accept)
			if x.hasContent() {
				root.addChild(x)
			}
		}
	}
	return root
}

func (n *Node) hasContent() bool {
	for _, c := range n.Children {
		if c.Leaf() {
			return true
		} else {
			if c.hasContent() {
				return true
			}
		}
	}

	return false
}
