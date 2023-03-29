package gotest

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	smallBufferSize  = 64
	mediumBufferSize = 4096
)

type NestedTest struct {
	Value    Test
	Children []NestedTest
	Log      []byte
}

type Set struct {
	Err   error
	Tests []NestedTest
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: bufio.NewScanner(r)}
}

type Reader struct {
	r *bufio.Scanner
}

func (r *Reader) ReadAll() (Set, error) {
	var errs []error

	prefix := &prefixNode{}

	for r.r.Scan() {
		line := r.r.Bytes()

		var row Entry
		if err := json.Unmarshal(line, &row); err != nil {
			errs = append(errs, fmt.Errorf("json.Unmarshal: %w", err))
		}

		if len(row.TestName) > 0 {
			key := row.Package + "/" + row.TestName

			tc, ok := prefix.find(key)
			if !ok {
				obj := &Test{
					Name:    row.TestName,
					Package: row.Package,
				}
				prefix.insert(obj)
				tc = obj
			}

			tc.Update(row)
		}
	}

	result := Set{
		Err: errors.Join(errs...),
	}

	testCases := make([]NestedTest, 0, len(prefix.Children))

	for _, nod := range prefix.Children {
		if tc, ok := r.walk(nod, newOutputBuffer(Ptr(make([]string, 0)), "", 0)); ok {
			testCases = append(testCases, tc)
		}
	}

	result.Tests = make([]NestedTest, len(testCases))
	copy(result.Tests, testCases)

	return result, nil
}

func (r *Reader) walk(node *prefixNode, offsetBuf *outputBuffer) (NestedTest, bool) {
	var testCase NestedTest

	if node == nil {
		return testCase, false
	}

	testCase.Value = *node.Value

	isResultActionRow := func(s string) bool {
		isGroupPrefix := strings.Contains(s, "---")
		isAction := strings.Contains(s, strings.ToUpper(ActionFail)) ||
			strings.Contains(s, strings.ToUpper(ActionPass)) ||
			strings.Contains(s, strings.ToUpper(ActionSkip))

		return isGroupPrefix && isAction
	}

	output := testCase.Value.Output
	for idx := range output {
		if isResultActionRow(output[idx]) {
			output[idx] = offsetBuf.prefix + output[idx]
		}
	}

	*offsetBuf.rows = append(*offsetBuf.rows, output...)

	testCase.Value.Output = testCase.Value.Output[:0]

	offsetBuf.incrPrefix()
	defer offsetBuf.decrPrefix()

	for _, nod := range node.Children {
		res1 := newOutputBuffer(offsetBuf.rows, offsetBuf.prefix, len(*offsetBuf.rows))
		res1.prefix = offsetBuf.prefix
		offsetBuf.next = res1

		if child, ok := r.walk(nod, res1); ok {
			testCase.Children = append(testCase.Children, child)
		}
	}

	copied := append([]string{}, (*offsetBuf.rows)[offsetBuf.offset:]...)
	mark := make([]string, 0)

	mx := 1<<32 - 1
	for i := len(copied) - 1; i >= 0; i-- {
		if isResultActionRow(copied[i]) {
			cnt := strings.Count(copied[i], "\t")
			if cnt < mx {
				mx = cnt
			}

			mark = append(mark, copied[i])
			copied = append(copied[:i], copied[i+1:]...)
		}
	}

	sort.Slice(
		mark, func(i, j int) bool {
			return strings.Count(mark[i], "\t") < strings.Count(mark[j], "\t")
		},
	)

	for _, row := range append(copied, mark...) {
		testCase.Log = append(testCase.Log, []byte(strings.Replace(row, "\t", "", mx))...)
	}

	return testCase, true
}

func newOutputBuffer(rows *[]string, prefix string, offset int) *outputBuffer {
	return &outputBuffer{rows: rows, prefix: prefix, offset: offset}
}

type outputBuffer struct {
	prefix string
	rows   *[]string
	offset int
	next   *outputBuffer
}

func (r *outputBuffer) incrPrefix() {
	r.prefix += "\t"
}

func (r *outputBuffer) decrPrefix() {
	r.prefix = strings.TrimSuffix(r.prefix, "\t")
}

func Ptr[T any](p T) *T {
	return &p
}
