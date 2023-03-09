package goallure

import (
	"strings"

	"github.com/robotomize/go-allure/internal/goexec"
)

type testPrefixNode struct {
	key      string
	value    *goexec.GoTest
	children []*testPrefixNode
}

func (t *testPrefixNode) find(key string) (*goexec.GoTest, bool) {
	for _, n := range t.children {
		if key == n.key {
			return n.value, true
		}

		if strings.HasPrefix(key, n.key) {
			return n.find(key)
		}
	}

	return nil, false
}

func (t *testPrefixNode) insert(obj *goexec.GoTest) {
	key := prefixIdxKey(obj.Package, obj.Name)

	if t.isChildExist(obj, key) {
		return
	}

	t.children = append(
		t.children, &testPrefixNode{
			key:   key,
			value: obj,
		},
	)
}

func (t *testPrefixNode) isChildExist(obj *goexec.GoTest, key string) bool {
	var curr *testPrefixNode
	var exist bool
	for idx, n := range t.children {
		if prefixIdxKey(obj.Package, obj.Name) == n.key {
			return true
		}

		if strings.HasPrefix(n.key, key) {
			if curr != nil {
				curr.children = append(curr.children, n)
				t.children = append(t.children[:idx], t.children[idx+1:]...)
				continue
			}
			curr = &testPrefixNode{
				key:      key,
				value:    obj,
				children: []*testPrefixNode{n},
			}
			t.children[idx] = curr
			exist = true
			continue
		}

		if prefixIdxKey(obj.Package, obj.Name) != n.key && strings.HasPrefix(key, n.key) {
			n.insert(obj)
			return true
		}
	}
	return exist
}

func prefixIdxKey(pkg, name string) string {
	return pkg + name
}
