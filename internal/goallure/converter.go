package goallure

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/robotomize/go-allure/internal/allure"
	"github.com/robotomize/go-allure/internal/goexec"
	"github.com/robotomize/go-allure/internal/slice"
)

type Attachment struct {
	Name   string
	Mime   string
	Source string
	Body   []byte
}

type Result struct {
	Err          error
	GoTestOutput io.Reader
	Attachments  []Attachment
	Tests        []allure.Test
}

type Option func(options *Options)

type Options struct {
	forceAttachment bool
	buildTags       []string
	allureLabels    []allure.Label
}

func WithForceAttachment() Option {
	return func(c *Options) {
		c.forceAttachment = true
	}
}

func WithBuildTags(tags ...string) Option {
	return func(options *Options) {
		options.buildTags = tags
	}
}

func WithAllureLabels(labels ...allure.Label) Option {
	return func(options *Options) {
		options.allureLabels = labels
	}
}

type AllureConverter interface {
	Output1(ctx context.Context) (Result, error)
}

func New(rootDir string, reader io.Reader, opts ...Option) AllureConverter {
	c := converter{
		rootDir:      rootDir,
		reader:       reader,
		rootTestNode: &testPrefixNode{},
		goTestFiles:  make(map[string]goexec.GoTestFile),
	}

	for _, o := range opts {
		o(&c.opts)
	}

	return &c
}

type converter struct {
	opts         Options
	rootDir      string
	reader       io.Reader
	rootTestNode *testPrefixNode

	goTestFiles map[string]goexec.GoTestFile
}

func (c *converter) Output1(ctx context.Context) (Result, error) {
	var errs []error
	var result Result
	const defaultBufSize = 4096

	outputBuf := bytes.NewBuffer(make([]byte, 0, defaultBufSize))

	scanner := bufio.NewScanner(c.reader)
	for scanner.Scan() {
		line := scanner.Bytes()

		var row goexec.GoTestEntry
		if err := json.Unmarshal(line, &row); err != nil {
			errs = append(errs, fmt.Errorf("json.Unmarshal: %w", err))
		}

		outputBuf.WriteString(fmt.Sprintf("%s\n", row.Output))
		c.add(row)
	}

	outputBuf.WriteByte('\n')
	result.GoTestOutput = outputBuf

	result.Err = errors.Join(errs...)

	modules, err := goexec.WalkModules(ctx, c.rootDir, c.opts.buildTags...)
	if err != nil {
		return Result{}, fmt.Errorf("goallure.WalkModules: %w", err)
	}

	goTestFiles, err := goexec.ParseTestFiles(ctx, modules)
	if err != nil {
		return Result{}, fmt.Errorf("goallure.ParseTestFiles: %w", err)
	}

	for _, goTestFile := range goTestFiles {
		id := prefixIdxKey(goTestFile.PackageName, goTestFile.TestName)
		c.goTestFiles[id] = goTestFile
	}

	hashFn := md5.New()

	hasher := func(b []byte) []byte {
		hashFn.Reset()
		hashFn.Write(b)

		return hashFn.Sum(nil)
	}

	hostname, _ := os.Hostname()
	for _, nod := range c.rootTestNode.children {
		goTest := nod.value

		id := uuid.New().String()
		status := c.convertStatus(goTest, goTest.Status)
		testCase := &allure.Test{
			UUID:        id,
			Name:        goTest.Name,
			Status:      status,
			Stage:       allure.StageFinished,
			Steps:       make([]*allure.Step, 0),
			Labels:      make([]allure.Label, 0),
			Parameters:  make([]allure.Parameter, 0),
			Attachments: make([]allure.Attachment, 0),
		}

		c.addLabels(goTest, testCase, hostname)

		testCaseID := hasher(
			[]byte(fmt.Sprintf(
				"%v", struct {
					Labels   []allure.Label
					FullName string
				}{testCase.Labels, testCase.FullName},
			)),
		)

		if goTest.Stop.Before(goTest.Start) {
			goTest.Stop = time.Now()
		}

		historyID := hasher(testCaseID)
		testCase.TestCaseID = hex.EncodeToString(testCaseID)
		testCase.HistoryID = hex.EncodeToString(historyID)
		testCase.Start = goTest.Start.UnixMilli()
		testCase.Stop = goTest.Stop.UnixMilli()

		c.prepareSteps(nod, testCase, &result.Attachments)
		c.addAttachment(goTest, testCase, &result.Attachments)

		result.Tests = append(result.Tests, *testCase)
	}

	return result, nil
}

func (c *converter) addLabels(goTest *goexec.GoTest, testCase *allure.Test, hostname string) {
	prefix := prefixIdxKey(goTest.Package, goTest.Name)
	goTestFile, ok := c.goTestFiles[prefix]
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

		testCase.FullName = fmt.Sprintf("%s/%s:%s", goTestFile.PackageName, goTestFile.FileName, goTest.Name)
	}

	testCase.Labels = append(testCase.Labels, c.opts.allureLabels...)
}

func (c *converter) addAttachment(
	goTest *goexec.GoTest,
	allureOutput any,
	attachments *[]Attachment,
) {
	hasAttachment := c.opts.forceAttachment || goTest.Status == goexec.GoTestActionPanic ||
		goTest.Status == goexec.GoTestActionFail

	if hasAttachment {
		var body []byte
		for _, row := range goTest.Output {
			body = append(body, append([]byte(row), '\n')...)
		}

		name := "stdout"
		mime := "plain/text"
		source := fmt.Sprintf("%s-attachment.txt", uuid.New().String())

		*attachments = append(
			*attachments, Attachment{
				Name:   name,
				Mime:   mime,
				Source: source,
				Body:   body,
			},
		)

		switch tc := allureOutput.(type) {
		case *allure.Test:
			tc.Attachments = append(
				tc.Attachments, allure.Attachment{
					Name:   name,
					Source: source,
					Type:   mime,
				},
			)
		case *allure.Step:
			tc.Attachments = append(
				tc.Attachments, allure.Attachment{
					Name:   name,
					Source: source,
					Type:   mime,
				},
			)
		default:
		}
	}
}

func (c *converter) prepareSteps(nod *testPrefixNode, allureTest *allure.Test, attachments *[]Attachment) {
	for _, node := range nod.children {
		goTest := node.value
		if goTest.Stop.Before(goTest.Start) {
			goTest.Stop = time.Now()
		}

		name := goTest.Name
		status := c.convertStatus(goTest, goTest.Status)

		step := &allure.Step{
			Name:        name,
			Status:      status,
			Stage:       allure.StageFinished,
			Start:       goTest.Start.UnixMilli(),
			Stop:        goTest.Stop.UnixMilli(),
			Steps:       make([]*allure.Step, 0),
			Attachments: make([]allure.Attachment, 0),
			Parameters:  make([]allure.Parameter, 0),
		}

		allureTest.Steps = append(allureTest.Steps, step)
		c.addAttachment(goTest, step, attachments)
		c.appendSteps(step, node, attachments)
	}
}

func (c *converter) add(entry goexec.GoTestEntry) {
	if len(entry.TestName) > 0 {
		key := prefixIdxKey(entry.Package, entry.TestName)
		tc, ok := c.rootTestNode.find(key)
		if !ok {
			obj := &goexec.GoTest{
				Name:    entry.TestName,
				Package: entry.Package,
			}
			c.rootTestNode.insert(obj)
			tc = obj
		}

		c.updateState(tc, entry)
	}
}

func (c *converter) appendSteps(s *allure.Step, n *testPrefixNode, attachments *[]Attachment) {
	for _, n1 := range n.children {
		goTestChildObj := n1.value
		if goTestChildObj.Stop.Before(goTestChildObj.Start) {
			goTestChildObj.Stop = time.Now()
		}

		name := goTestChildObj.Name
		status := c.convertStatus(goTestChildObj, goTestChildObj.Status)

		step := &allure.Step{
			Name:        name,
			Status:      status,
			Stage:       allure.StageFinished,
			Start:       goTestChildObj.Start.UnixMilli(),
			Stop:        goTestChildObj.Stop.UnixMilli(),
			Steps:       make([]*allure.Step, 0),
			Attachments: make([]allure.Attachment, 0),
			Parameters:  make([]allure.Parameter, 0),
		}

		s.Steps = append(s.Steps, step)

		c.appendSteps(step, n1, attachments)
	}
}

func (*converter) updateState(goTest *goexec.GoTest, row goexec.GoTestEntry) {
	switch row.Action {
	case goexec.GoTestActionCont:
		goTest.Stage = goexec.GoTestActionCont
	case goexec.GoTestActionSkip:
		goTest.Stop = row.Time
		goTest.Status = goexec.GoTestActionSkip
		goTest.Stage = goexec.GoTestActionSkip
		goTest.Elapsed = goTest.Stop.Sub(goTest.Start)
	case goexec.GoTestActionFail:
		goTest.Stop = row.Time
		goTest.Status = goexec.GoTestActionFail
		goTest.Stage = goexec.GoTestActionFail
		goTest.Elapsed = goTest.Stop.Sub(goTest.Start)
	case goexec.GoTestActionOutput:
		goTest.Output = append(goTest.Output, row.Output)
	case goexec.GoTestActionPass:
		goTest.Stop = row.Time
		goTest.Status = goexec.GoTestActionPass
		goTest.Stage = goexec.GoTestActionPass
		goTest.Elapsed = goTest.Stop.Sub(goTest.Start)
	case goexec.GoTestActionPause:
		goTest.Stage = goexec.GoTestActionPause
	case goexec.GoTestActionRun:
		goTest.Start = row.Time
		goTest.Stage = goexec.GoTestActionRun
	}
	goTest.Output = append(goTest.Output, row.Output)
}

func (*converter) convertStatus(goTest *goexec.GoTest, status string) string {
	var allureStage string
	switch status {
	case goexec.GoTestActionSkip:
		allureStage = allure.StatusSkip
	case goexec.GoTestActionFail:
		if _, ok := slice.Find(
			goTest.Output, func(v string) bool {
				return strings.Contains(v, "panic")
			},
		); ok {
			allureStage = allure.StatusBroken
			break
		}
		allureStage = allure.StatusFail
	case goexec.GoTestActionPass:
		allureStage = allure.StatusPass
	default:
		allureStage = allure.StatusBroken
	}

	return allureStage
}
