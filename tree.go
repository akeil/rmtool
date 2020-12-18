package rm

import (
	"fmt"
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

func (n *Node) IsLeaf() bool {
	return n.Type() != CollectionType
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
		nodes = append(nodes, newNode(id, m))
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
