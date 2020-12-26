package rm

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Node is the representation for an entry in the content tree.
// A not can either be a document or a collection (which has child nodes).
type Node struct {
	Meta
	// ParentNode holds a reference to the parent or nul.
	ParentNode *Node
	// Children is a list of all child nodes.
	Children []*Node
}

func newNode(m Meta) *Node {
	return &Node{
		Meta:     m,
		Children: make([]*Node, 0),
	}
}

// tell if this node is lovated in the root folder.
func (n *Node) inRoot() bool {
	return n.ID() == ""
}

// Leaf tells if this is a leaf node (without children).
func (n *Node) Leaf() bool {
	return n.Type() != CollectionType && !n.inRoot()
}

// Path returns the path components for this node.
// That is, the IDs of its parent and grandparent up to the root node.
func (n *Node) Path() []string {
	p := make([]string, 0)

	ctx := n
	for {
		if ctx.ParentNode == nil {
			break
		}
		p = append(p, ctx.ParentNode.ID())
		ctx = ctx.ParentNode
	}

	return p
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
	child.ParentNode = n
}

// Put attempts to accomodate this node in the subtree starting at this node.
// The child node can be added as an immediate child or grandchild.
// Returns `true` if the node could be added to the tree.
func (n *Node) put(other *Node) bool {
	if other.Parent() == n.ID() {
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

// BuildTree creates a tree view of all items in the given repository.
// Returns the root node.
func BuildTree(r Repository) (*Node, error) {
	items, err := r.List()
	if err != nil {
		return nil, err
	}

	root := newNode(nodeMeta{name: "root"})
	root.addChild(newNode(nodeMeta{
		id:   "trash",
		name: "Trash",
	}))

	nodes := make([]*Node, len(items))
	for i, item := range items {
		nodes[i] = newNode(item)
	}

	// build a tree structure from the flat list
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
		return nil, fmt.Errorf("could not fit all notes into the tree")
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
	if one.ID() == "trash" {
		return false
	} else if other.ID() == "trash" {
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
		return one.ID() < other.ID()
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
func (n *Node) Filtered(match NodeFilter) *Node {
	root := newNode(n.Meta)
	for _, c := range n.Children {
		if c.Leaf() {
			if match(c) {
				root.addChild(newNode(c.Meta))
			}
		} else {
			x := c.Filtered(match)
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

// implements the Meta interface for "virtual" nodes
// (root and "trash").
type nodeMeta struct {
	id     string
	parent string
	name   string
}

func (n nodeMeta) ID() string {
	return n.id
}

func (n nodeMeta) Version() uint {
	return uint(1)
}

func (n nodeMeta) Name() string {
	return n.name
}

func (n nodeMeta) SetName(s string) {}

func (n nodeMeta) Type() NotebookType {
	return CollectionType
}

func (n nodeMeta) Pinned() bool {
	return false
}

func (n nodeMeta) SetPinned(b bool) {}

func (n nodeMeta) LastModified() time.Time {
	return time.Time{}
}

func (n nodeMeta) Parent() string {
	return n.Parent()
}

func (n nodeMeta) Reader(path ...string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented for virtual nodes")
}

func (n nodeMeta) PagePrefix(id string, index int) string {
	return ""
}
