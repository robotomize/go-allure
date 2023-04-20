package gotest

import (
	"strings"
)

type prefixNode struct {
	Key      string
	Value    *Test
	Children []*prefixNode
}

// find - find test case from tree.
func (t *prefixNode) find(key string) (*Test, bool) {
	// find Iterate through each child of the current node.
	// Check if the child's key matches the given key.
	for _, n := range t.Children {
		if key == n.Key {
			return n.Value, true
		}

		// Check if the given key is a subtest of the child's key.
		// If it is, recursively call the find function on the child.
		if t.isSubTest(key, n.Key) {
			return n.find(key)
		}
	}

	return nil, false
}

func (t *prefixNode) isSubTest(key, nodeKey string) bool {
	return strings.HasPrefix(key, nodeKey) && strings.Count(key, "/") != strings.Count(nodeKey, "/")
}

// insert - insert new test case to the tree.
func (t *prefixNode) insert(obj *Test) {
	// Generate a key for the Test based on its package and name fields using the prefixIdxKey function.
	key := t.prefixIdxKey(obj.Package, obj.Name)

	// Check if a child already exists with the same key. If one does, return without doing anything.
	if t.isChildExist(obj, key) {
		return
	}

	// Otherwise, append a new child node with the generated key and Test pointer to the Children field slice.
	t.Children = append(
		t.Children, &prefixNode{
			Key:   key,
			Value: obj,
		},
	)
}

// isChildExist - does the current node have a child?
func (t *prefixNode) isChildExist(obj *Test, key string) bool {
	var curr *prefixNode
	var exist bool

	// Iterate through each child in the current node's Children slice.
	for idx, n := range t.Children {
		// If a child already exists with the same key as the generated key for the Test, return true.
		if t.prefixIdxKey(obj.Package, obj.Name) == n.Key {
			return true
		}

		// If the key is a subtest of a child's key, create a new child node that combines the two and add it to the Children slice.
		if strings.HasPrefix(n.Key, key) {
			if curr != nil {
				curr.Children = append(curr.Children, n)
				t.Children = append(t.Children[:idx], t.Children[idx+1:]...)
				continue
			}
			curr = &prefixNode{
				Key:      key,
				Value:    obj,
				Children: []*prefixNode{n},
			}
			t.Children[idx] = curr
			exist = true
			continue
		}

		// If the generated key for the Test is not equal to a child's key, but is a prefix of it, recursively call the insert function on the child.
		if t.prefixIdxKey(obj.Package, obj.Name) != n.Key && strings.HasPrefix(key, n.Key) {
			if strings.Count(key, "/") == strings.Count(n.Key, "/") {
				continue
			}
			n.insert(obj)
			return true
		}
	}
	return exist
}

// Define the prefixIdxKey function on the prefixNode struct that takes in a package and name string and returns a key string.
// Concatenate the package and name strings together with a "/" separator.
func (t *prefixNode) prefixIdxKey(pkg, name string) string {
	return pkg + "/" + name
}
