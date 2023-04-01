package gotest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/robotomize/go-allure/internal/slice"
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

func (r *Reader) ReadAll(ctx context.Context) (Set, error) {
	var errs []error

	prefix := &prefixNode{}

	for r.r.Scan() {
		select {
		case <-ctx.Done():
			return Set{}, ctx.Err()
		default:
		}

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
		Err: errors.Join(errs...), // nolint
	}

	testCases := make([]NestedTest, 0, len(prefix.Children))

	for _, nod := range prefix.Children {
		if tc, ok := r.walk(nod, newPrefixLog()); ok {
			testCases = append(testCases, tc)
		}
	}

	result.Tests = make([]NestedTest, len(testCases))
	copy(result.Tests, testCases)

	return result, nil
}

func (r *Reader) walk(node *prefixNode, prefix *prefixLog) (NestedTest, bool) {
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
			output[idx] = prefix.prefix + output[idx]
		}
		prefix.buf.WriteString(output[idx])
	}

	testCase.Value.Output = testCase.Value.Output[:0]

	prefix.incrPrefix()
	defer prefix.decrPrefix()

	for _, nod := range node.Children {
		if child, ok := r.walk(nod, prefix.copy()); ok {
			testCase.Children = append(testCase.Children, child)
		}
	}

	reader := bytes.NewReader(prefix.buf.Bytes())
	if _, err := reader.Seek(int64(prefix.pos), io.SeekCurrent); err != nil {
		return NestedTest{}, false
	}

	all, err := io.ReadAll(reader)
	if err != nil {
		return NestedTest{}, false
	}

	copied := slice.Map(
		bytes.Split(all, []byte{'\n'}), func(t []byte) string {
			return string(t) + "\n"
		},
	)

	lastIdx := len(copied) - 1
	copied[lastIdx] = strings.TrimSuffix(copied[lastIdx], "\n")

	mark := make([]string, 0)

	mx := 1<<31 - 1
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

	log := make([]byte, 0, len(copied)+len(mark))
	for _, row := range append(copied, mark...) {
		log = append(log, []byte(strings.Replace(row, "\t", "", mx))...)
	}

	testCase.Log = append(log, '\n')

	return testCase, true
}

func newPrefixLog() *prefixLog {
	return &prefixLog{buf: bytes.NewBuffer(make([]byte, 0, 64))}
}

type prefixLog struct {
	prefix string
	buf    *bytes.Buffer
	pos    int
}

func (r *prefixLog) copy() *prefixLog {
	r1 := newPrefixLog()
	r1.buf = r.buf
	r1.prefix = r.prefix
	r1.pos = r.buf.Len()
	return r1
}

func (r *prefixLog) incrPrefix() {
	r.prefix += "\t"
}

func (r *prefixLog) decrPrefix() {
	r.prefix = strings.TrimSuffix(r.prefix, "\t")
}
