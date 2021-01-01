package rm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalkTree(t *testing.T) {
	assert := assert.New(t)
	root := sampleTree()

	actual := make([]string, 0)
	root.Walk(func(n *Node) error {
		actual = append(actual, n.ID())
		return nil
	})

	expected := []string{"root", "a0", "a1", "a2", "b0", "c0", "c1", "c2"}
	assert.ElementsMatch(expected, actual, "not every node was visited")

	// error from visitor function should be returned
	expectedErr := errors.New("some error")
	// and walk should abort on error
	counter := 0
	actualErr := root.Walk(func(n *Node) error {
		counter++
		return expectedErr
	})
	assert.Equal(expectedErr, actualErr)
	assert.Equal(1, counter)
}

func TestMatchName(t *testing.T) {
	assert := assert.New(t)
	n := node("foobar", "Foo Bar", DocumentType)

	assert.True(MatchName("Foo Bar")(n), "exact match")
	assert.True(MatchName("foo")(n), "partial match")
	assert.True(MatchName("bar")(n), "partial match")
	assert.True(MatchName("oo")(n), "partial match")

	assert.False(MatchName("not foo")(n), "no match")

	assert.True(MatchName("")(n), "empty string should match all")
}

func TestMatchType(t *testing.T) {
	assert := assert.New(t)

	doc := node("foo", "Foo", DocumentType)
	assert.True(IsDocument(doc))
	assert.False(IsFolder(doc))

	folder := node("bar", "Bar", CollectionType)
	assert.False(IsDocument(folder))
	assert.True(IsFolder(folder))
}

func TestFilterTree(t *testing.T) {
	assert := assert.New(t)
	root := sampleTree()

	// return matching docs AND folders on path up to root
	f := root.Filtered(MatchName("c"))
	assert.Equal("root", f.ID(), "root node was lost in filter")
	assert.Equal(1, len(f.Children), "intermediate folder lost in filter")
	assert.Equal(3, len(f.Children[0].Children), "matching nodes incomplete")

	// multiple matches
	f = root.Filtered(IsDocument, MatchName("0"))
	assert.Equal(2, len(f.Children))
	assert.Equal("a0", f.Children[0].ID())
	assert.Equal("b0", f.Children[1].ID())
	assert.Equal(1, len(f.Children[1].Children))
	assert.Equal("c0", f.Children[1].Children[0].ID())
}

func TestSortTree(t *testing.T) {
	assert := assert.New(t)
	root := sampleTree()
	assert.Equal(root.Children[0].ID(), "a0", "precondition failed")
	assert.Equal(root.Children[3].ID(), "b0", "precondition failed")

	root.Sort(DefaultSort)
	assert.Equal(root.Children[0].ID(), "b0", "Folders must come before documents")
	assert.Equal(root.Children[1].ID(), "a0", "documents by name")
}

func TestTreePath(t *testing.T) {
	assert := assert.New(t)
	root := sampleTree()

	assert.Equal(root.Path(), []string{})
	assert.Equal(root.Children[0].Path(), []string{"root"})
	assert.Equal(root.Children[3].Children[0].Path(), []string{"root", "b0"})

	// filter and sort should not disturb paths of remaining elements
	root = root.Filtered(MatchName("c0"))
	root.Sort(DefaultSort)
	assert.Equal(root.Path(), []string{})
	assert.Equal(root.Children[0].Path(), []string{"root"})
	assert.Equal(root.Children[0].Children[0].Path(), []string{"root", "b0"})
}

// creates a tree like this:
//
// root
// |
// +- a0
// +- a1
// +- a2
// ´- b0
//    |
//    +- c0
//    +- c1
//    ´- c2
func sampleTree() *Node {
	root := node("root", "root", CollectionType)

	a := []string{"a0", "a1", "a2"}
	for _, id := range a {
		root.addChild(node(id, id, DocumentType))
	}

	b := node("b0", "b0", CollectionType)
	root.addChild(b)
	c := []string{"c0", "c1", "c2"}
	for _, id := range c {
		b.addChild(node(id, id, DocumentType))
	}

	return root
}

func node(id, name string, t NotebookType) *Node {
	return newNode(&nodeMeta{id, "", name, t})
}
