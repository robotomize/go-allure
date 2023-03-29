package gotest

import (
	"strings"
)

type prefixNode struct {
	Key      string
	Value    *Test
	Children []*prefixNode
}

func (t *prefixNode) find(key string) (*Test, bool) {
	for _, n := range t.Children {
		if key == n.Key {
			return n.Value, true
		}

		if strings.HasPrefix(key, n.Key) {
			if strings.Count(key, "/") != strings.Count(n.Key, "/") {
				return n.find(key)
			}

			return n.Value, true
		}
	}

	return nil, false
}

func (t *prefixNode) insert(obj *Test) {
	key := t.prefixIdxKey(obj.Package, obj.Name)

	if t.isChildExist(obj, key) {
		return
	}

	t.Children = append(
		t.Children, &prefixNode{
			Key:   key,
			Value: obj,
		},
	)
}

func (t *prefixNode) isChildExist(obj *Test, key string) bool {
	var curr *prefixNode
	var exist bool
	for idx, n := range t.Children {
		if t.prefixIdxKey(obj.Package, obj.Name) == n.Key {
			return true
		}

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

		if t.prefixIdxKey(obj.Package, obj.Name) != n.Key && strings.HasPrefix(key, n.Key) {
			n.insert(obj)
			return true
		}
	}
	return exist
}

func (t *prefixNode) prefixIdxKey(pkg, name string) string {
	return pkg + "/" + name
}
