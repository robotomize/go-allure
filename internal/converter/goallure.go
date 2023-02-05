package converter

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robotomize/go-allure/internal/slice"

	"github.com/robotomize/go-allure/internal/allure"
	"github.com/robotomize/go-allure/internal/gointernal"
)

type Option func(*converter)

type AllureConverter interface {
	Append(row gointernal.GoTestLogEntry)
	Output() []allure.Test
}

func New(goTestFiles []gointernal.GoTestFile, opts ...Option) AllureConverter {
	c := converter{
		prefixTree:  &simplePrefixNode{},
		goTestFiles: make(map[string]gointernal.GoTestFile),
	}

	for _, o := range opts {
		o(&c)
	}

	for _, goTestFile := range goTestFiles {
		c.goTestFiles[prefixIdxKey(goTestFile.PackageName, goTestFile.TestName)] = goTestFile
	}

	return &c
}

type converter struct {
	prefixTree  *simplePrefixNode
	goTestFiles map[string]gointernal.GoTestFile
}

func (c *converter) Append(row gointernal.GoTestLogEntry) {
	if row.TestName == "" {
		return
	}

	key := prefixIdxKey(row.Package, row.TestName)
	testCase, ok := c.prefixTree.find(key)
	if !ok {
		obj := &gointernal.GoTest{
			Name:    row.TestName,
			Package: row.Package,
		}
		c.prefixTree.insert(obj)
		testCase = obj
	}

	c.updateGoTestState(testCase, row)
}

func (c *converter) Output() []allure.Test {
	hashFunc := md5.New()

	createTestCaseID := func(labels []allure.Label, fullName string) []byte {
		hashFunc.Reset()
		hashFunc.Write(
			[]byte(fmt.Sprintf(
				"%v", struct {
					Labels   []allure.Label
					FullName string
				}{labels, fullName},
			)),
		)
		return hashFunc.Sum(nil)
	}

	createHistoryID := func(b []byte) []byte {
		hashFunc.Reset()
		hashFunc.Write(b)
		return hashFunc.Sum(nil)
	}

	hostname, _ := os.Hostname()
	testCases := make([]allure.Test, 0, len(c.prefixTree.children))
	for _, nod := range c.prefixTree.children {
		goTestObj := nod.value

		testCase := allure.Test{
			UUID:        uuid.New(),
			Status:      c.convertTestStatus(goTestObj, goTestObj.Status),
			Stage:       allure.StageFinished,
			Name:        goTestObj.Name,
			Steps:       make([]*allure.Step, 0),
			Labels:      make([]allure.Label, 0),
			Parameters:  make([]allure.Parameter, 0),
			Attachments: make([]allure.Attachment, 0),
		}

		goTestFile, ok := c.goTestFiles[prefixIdxKey(goTestObj.Package, goTestObj.Name)]
		if ok {
			testCase.Labels = []allure.Label{
				{
					Name:  "package",
					Value: goTestFile.PackageName,
				},
				{
					Name:  "testClass",
					Value: goTestFile.PackageName + "/" + goTestFile.TestName,
				},
				{
					Name:  "testMethod",
					Value: goTestFile.TestName,
				},
				{
					Name:  "language",
					Value: "golang",
				},
				{
					Name:  "go-version",
					Value: goTestFile.GoVersion,
				},
				{
					Name:  "host",
					Value: hostname,
				},
			}
			testCase.FullName = fmt.Sprintf("%s/%s:%s", goTestFile.PackageName, goTestFile.FileName, goTestObj.Name)
		}

		testCaseID := createTestCaseID(testCase.Labels, testCase.FullName)

		if goTestObj.Stop.Before(goTestObj.Start) {
			goTestObj.Stop = time.Now()
		}

		testCase.TestCaseID = hex.EncodeToString(testCaseID)
		testCase.HistoryID = hex.EncodeToString(createHistoryID(testCaseID))
		testCase.Start = goTestObj.Start.UnixMilli()
		testCase.Stop = goTestObj.Stop.UnixMilli()

		for _, stepNode := range nod.children {
			goTestStepObj := stepNode.value
			if goTestStepObj.Stop.Before(goTestStepObj.Start) {
				goTestStepObj.Stop = time.Now()
			}
			step := &allure.Step{
				Name:        goTestStepObj.Name,
				Status:      c.convertTestStatus(goTestStepObj, goTestStepObj.Status),
				Stage:       allure.StageFinished,
				Start:       goTestStepObj.Start.UnixMilli(),
				Stop:        goTestStepObj.Stop.UnixMilli(),
				Steps:       make([]*allure.Step, 0),
				Attachments: make([]allure.Attachment, 0),
				Parameters:  make([]allure.Parameter, 0),
			}
			testCase.Steps = append(testCase.Steps, step)

			c.appendSteps(step, stepNode)
		}

		testCases = append(testCases, testCase)
	}

	return testCases
}

func (c *converter) appendSteps(s *allure.Step, n *simplePrefixNode) {
	for _, n1 := range n.children {
		goTestChildObj := n1.value
		if goTestChildObj.Stop.Before(goTestChildObj.Start) {
			goTestChildObj.Stop = time.Now()
		}

		step := &allure.Step{
			Name:        goTestChildObj.Name,
			Status:      c.convertTestStatus(goTestChildObj, goTestChildObj.Status),
			Stage:       allure.StageFinished,
			Start:       goTestChildObj.Start.UnixMilli(),
			Stop:        goTestChildObj.Stop.UnixMilli(),
			Steps:       make([]*allure.Step, 0),
			Attachments: make([]allure.Attachment, 0),
			Parameters:  make([]allure.Parameter, 0),
		}

		s.Steps = append(s.Steps, step)

		c.appendSteps(step, n1)
	}
}

func (c *converter) updateGoTestState(g *gointernal.GoTest, row gointernal.GoTestLogEntry) {
	switch row.Action {
	case gointernal.GoTestActionCont:
		g.Stage = gointernal.GoTestActionCont
	case gointernal.GoTestActionSkip:
		g.Stop = row.Time
		g.Status = gointernal.GoTestActionSkip
		g.Stage = gointernal.GoTestActionSkip
		g.Elapsed = g.Stop.Sub(g.Start)
	case gointernal.GoTestActionFail:
		g.Stop = row.Time
		g.Status = gointernal.GoTestActionFail
		g.Stage = gointernal.GoTestActionFail
		g.Elapsed = g.Stop.Sub(g.Start)
	case gointernal.GoTestActionOutput:
		g.Output = append(g.Output, row.Output)
	case gointernal.GoTestActionPass:
		g.Stop = row.Time
		g.Status = gointernal.GoTestActionPass
		g.Stage = gointernal.GoTestActionPass
		g.Elapsed = g.Stop.Sub(g.Start)
	case gointernal.GoTestActionPause:
		g.Stage = gointernal.GoTestActionPause
	case gointernal.GoTestActionRun:
		g.Start = row.Time
		g.Stage = gointernal.GoTestActionRun
	}
	g.Output = append(g.Output, row.Output)
}

func (c *converter) convertTestStatus(g *gointernal.GoTest, stage string) string {
	var allureStage string
	switch stage {
	case gointernal.GoTestActionSkip:
		allureStage = allure.StatusSkip
	case gointernal.GoTestActionFail:
		if _, ok := slice.Find(
			g.Output, func(v string) bool {
				return strings.Contains(v, "panic")
			},
		); ok {
			allureStage = allure.StatusBroken
			break
		}
		allureStage = allure.StatusFail
	case gointernal.GoTestActionPass:
		allureStage = allure.StatusPass
	default:
		allureStage = allure.StatusBroken
	}

	return allureStage
}

type simplePrefixNode struct {
	key      string
	value    *gointernal.GoTest
	children []*simplePrefixNode
}

func (t *simplePrefixNode) find(key string) (*gointernal.GoTest, bool) {
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

func (t *simplePrefixNode) insert(obj *gointernal.GoTest) {
	key := prefixIdxKey(obj.Package, obj.Name)

	if t.isChildExist(obj, key) {
		return
	}

	t.children = append(
		t.children, &simplePrefixNode{
			key:   key,
			value: obj,
		},
	)
}

func (t *simplePrefixNode) isChildExist(obj *gointernal.GoTest, key string) bool {
	var curr *simplePrefixNode
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
			curr = &simplePrefixNode{
				key:      key,
				value:    obj,
				children: []*simplePrefixNode{n},
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
