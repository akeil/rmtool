package rm

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"akeil.net/akeil/rm/internal/logging"
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

// tell if this node is the root folder.
func (n *Node) isRoot() bool {
	return n.ID() == "root"
}

// IsLeaf tells if this is a leaf node (without children).
func (n *Node) IsLeaf() bool {
	return n.Type() != CollectionType && !n.isRoot()
}

// Path returns the path components for this node.
// That is, the names of its parent and grandparent up to the root node.
func (n *Node) Path() []string {
	p := make([]string, 0)

	ctx := n
	for {
		if ctx.ParentNode == nil {
			break
		}
		// We are moving "up" towards root, but the path should start at root.
		// Therfore, *prepend* each item to the list.
		p = append([]string{ctx.ParentNode.Name()}, p...)
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
func BuildTree(items []Meta) *Node {
	root := newNode(&nodeMeta{name: "root", nbType: CollectionType})
	root.addChild(newNode(&nodeMeta{
		id:     "trash",
		name:   "Trash",
		nbType: CollectionType,
	}))

	nodes := make([]*Node, len(items))
	for i, item := range items {
		nodes[i] = newNode(item)
	}

	// build a tree structure from the flat list
	var change bool
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
		if !change {
			break
		}
	}

	if len(nodes) != 0 {
		logging.Warning("could not fit all notes into the tree")
	}

	return root
}

// NodeComparator is used to sort nodes in a tree.
// It should return true if "one" comes before "other".
type NodeComparator func(one, other *Node) bool

// Sort sorts the subtree starting at this node by the given sort rule.
// Sorting is in-place.
func (n *Node) Sort(compare NodeComparator) {
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
	if one.IsLeaf() && !other.IsLeaf() {
		return false
	} else if other.IsLeaf() && !one.IsLeaf() {
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
// The subtree will contain only nodes that match the given NodeFilter
// and the parent folders of the matched nodes.
func (n *Node) Filtered(match ...NodeFilter) *Node {

	matches := func(n *Node) bool {
		for _, accept := range match {
			if !accept(n) {
				return false
			}
		}
		return true
	}

	root := newNode(n.Meta)
	for _, child := range n.Children {
		if child.IsLeaf() {
			if matches(child) {
				root.addChild(newNode(child.Meta))
			}
		} else {
			x := child.Filtered(match...)
			// match against unfiltered `child`, allows path matches
			// but *add* the filtered child
			if x.hasContent() || matches(child) {
				root.addChild(x)
			}
		}
	}
	return root
}

func (n *Node) hasContent() bool {
	for _, c := range n.Children {
		if c.IsLeaf() {
			return true
		} else if c.hasContent() {
			return true
		}
	}

	return false
}

// MatchName creates a node filter that matches the given string against
// the Name of a node. The match is case insensitive and allows partial matches
// ("doc" matches "My Document").
func MatchName(s string) NodeFilter {
	return func(n *Node) bool {
		return strings.Contains(strings.ToLower(n.Name()), strings.ToLower(s))
	}
}

// MatchPath creates a node filter that matches on the path components of
// a node (case insensitive).
//
// The path to match against is expected to contain the item name,
// i.e. "foo/bar/baz" will match the item named "baz" in the folder "foo/bar".
func MatchPath(path string) NodeFilter {
	fragments := strings.Split(path, "/")
	match := make([]string, 0)
	for _, s := range fragments {
		// Remove empty components essentially clears leading and trailing "/"
		// and normalizes multiple "//" to single "/"
		if s != "" {
			match = append(match, s)
		}
	}
	var name string
	if len(match) > 0 {
		name = match[len(match)-1]
		match = match[:len(match)-1]
	}

	return func(n *Node) bool {
		// Edge case:
		// When path is "" or "/", match the root folder
		if len(match) == 0 && name == "" {
			return n.isRoot()
		}

		if !strings.EqualFold(name, n.Name()) {
			return false
		}

		p := n.Path()
		p = p[1:] // drop root element
		if len(p) != len(match) {
			return false
		}

		for i := 0; i < len(match); i++ {
			if !strings.EqualFold(p[i], match[i]) {
				return false
			}
		}
		return true
	}
}

// IsDocument is a Node filter that matches only documents (not foldeers).
func IsDocument(n *Node) bool {
	return n.Type() == DocumentType
}

// IsFolder is a node filter that matches only folders.
func IsFolder(n *Node) bool {
	return n.Type() == CollectionType
}

// IsPinned is a node filter that matches only pinned items.
func IsPinned(n *Node) bool {
	return n.Pinned()
}

// implements the Meta interface for "virtual" nodes
// (root and "trash").
type nodeMeta struct {
	id     string
	parent string
	name   string
	nbType NotebookType
}

func (n *nodeMeta) ID() string {
	return n.id
}

func (n *nodeMeta) Version() uint {
	return uint(1)
}

func (n *nodeMeta) Name() string {
	return n.name
}

func (n *nodeMeta) SetName(s string) {}

func (n *nodeMeta) Type() NotebookType {
	return n.nbType
}

func (n *nodeMeta) Pinned() bool {
	return false
}

func (n *nodeMeta) SetPinned(b bool) {}

func (n *nodeMeta) LastModified() time.Time {
	return time.Time{}
}

func (n *nodeMeta) Parent() string {
	return n.parent
}

func (n *nodeMeta) Reader(path ...string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented for virtual nodes")
}

func (n *nodeMeta) PagePrefix(id string, index int) string {
	return ""
}

func (n *nodeMeta) Validate() error {
	return nil
}
