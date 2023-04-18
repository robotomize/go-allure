package exporter

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/robotomize/go-allure/internal/allure"
	"github.com/robotomize/go-allure/internal/gotest"
	"github.com/robotomize/go-allure/internal/parser"
	"github.com/robotomize/go-allure/internal/slice"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()
}

type Attachment struct {
	Name   string
	Mime   string
	Source string
	Body   []byte
}

type Report struct {
	Err         error
	OutputLog   io.Reader
	Attachments []Attachment
	Tests       []allure.Test
}

type Option func(options *Options)

type Options struct {
	forceAttachment bool
	allureLabels    []allure.Label
}

func WithForceAttachment() Option {
	return func(c *Options) {
		c.forceAttachment = true
	}
}

func WithAllureLabels(labels ...allure.Label) Option {
	return func(options *Options) {
		options.allureLabels = labels
	}
}

type Reader interface {
	ReadAll(ctx context.Context) (gotest.Set, error)
}

type FileParser interface {
	ParseFiles(ctx context.Context) ([]parser.GoTestFile, error)
}

type AllureExporter interface {
	Read(ctx context.Context) error
	Export() (Report, error)
}

func New(fileParser FileParser, reader Reader, opts ...Option) AllureExporter {
	c := exporter{
		stdinReader: reader,
		fileParser:  fileParser,
		files:       make(map[string]parser.GoTestFile),
	}

	for _, o := range opts {
		o(&c.opts)
	}

	return &c
}

type exporter struct {
	opts        Options
	readErr     error
	fileParser  FileParser
	stdinReader Reader
	tests       []gotest.NestedTest
	files       map[string]parser.GoTestFile
}

func (e *exporter) Read(ctx context.Context) error {
	files, err := e.fileParser.ParseFiles(ctx)
	if err != nil {
		return fmt.Errorf("go parser ParseFiles: %w", err)
	}

	for _, file := range files {
		key := file.PackageName + file.TestName
		e.files[key] = file
	}

	all, err := e.stdinReader.ReadAll(ctx)
	if err != nil {
		return fmt.Errorf("stdin reader ReadAll: %w", err)
	}

	e.tests = make([]gotest.NestedTest, len(all.Tests))
	copy(e.tests, all.Tests)
	e.readErr = all.Err

	return nil
}

func (e *exporter) Export() (Report, error) {
	var result Report

	const sampleBufferSize = 4096

	logBuf := bytes.NewBuffer(make([]byte, 0, sampleBufferSize))
	result.OutputLog = logBuf

	attachmentCh := make(chan Attachment)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	attachments := make([]Attachment, 0)
	defer func() {
		result.Attachments = append(result.Attachments, attachments...)
	}()

	go func() {
		defer wg.Done()
		for attachment := range attachmentCh {
			attachments = append(attachments, attachment)
		}
	}()

	hashFn := md5.New()

	hasher := func(b []byte) []byte {
		hashFn.Reset()
		hashFn.Write(b)

		return hashFn.Sum(nil)
	}

	for _, testCase := range e.tests {
		goTest := testCase.Value

		id := uuid.New().String()
		status := e.convertStatus(goTest)

		allureTestCase := allure.Test{
			UUID:        id,
			Name:        goTest.Name,
			Status:      status,
			Stage:       allure.StageFinished,
			Steps:       make([]allure.Step, 0),
			Labels:      make([]allure.Label, 0),
			Parameters:  make([]allure.Parameter, 0),
			Attachments: make([]allure.Attachment, 0),
		}

		e.defaultLabels(goTest, &allureTestCase)

		testCaseID := hasher([]byte(allureTestCase.FullName))

		if goTest.Stop.Before(goTest.Start) {
			goTest.Stop = time.Now()
		}

		historyID := hasher(testCaseID)
		allureTestCase.TestCaseID = hex.EncodeToString(testCaseID)
		allureTestCase.HistoryID = hex.EncodeToString(historyID)
		allureTestCase.Start = goTest.Start.UnixMilli()
		allureTestCase.Stop = goTest.Stop.UnixMilli()

		hasAttachment := e.opts.forceAttachment || goTest.Status == gotest.ActionPanic || goTest.Status == gotest.ActionFail
		if hasAttachment {
			source := fmt.Sprintf("%s-attachment.txt", uuid.New().String())
			mime := "plain/text"
			attachmentCh <- Attachment{
				Name:   goTest.Name,
				Mime:   mime,
				Source: source,
				Body:   testCase.Log,
			}

			allureTestCase.Attachments = append(
				allureTestCase.Attachments, allure.Attachment{
					Name:   goTest.Name,
					Source: source,
					Type:   "plain/text",
				},
			)
		}

		e.addStep(&allureTestCase, testCase, attachmentCh)
		result.Tests = append(result.Tests, allureTestCase)

		logBuf.Write(testCase.Log)
	}

	close(attachmentCh)

	wg.Wait()

	return result, nil
}

func (e *exporter) addStep(allureObj any, testCase gotest.NestedTest, ch chan<- Attachment) {
	for _, tc := range testCase.Children {
		goTest := tc.Value
		if goTest.Stop.Before(goTest.Start) {
			goTest.Stop = time.Now()
		}

		name := goTest.Name
		status := e.convertStatus(goTest)

		step := allure.Step{
			Name:        name,
			Status:      status,
			Stage:       allure.StageFinished,
			Start:       goTest.Start.UnixMilli(),
			Stop:        goTest.Stop.UnixMilli(),
			Steps:       make([]allure.Step, 0),
			Attachments: make([]allure.Attachment, 0),
			Parameters:  make([]allure.Parameter, 0),
		}

		hasAttachment := e.opts.forceAttachment || goTest.Status == gotest.ActionPanic || goTest.Status == gotest.ActionFail
		if hasAttachment {
			source := fmt.Sprintf("%s-attachment.txt", uuid.New().String())
			mime := "plain/text"

			ch <- Attachment{
				Name:   goTest.Name,
				Mime:   mime,
				Source: source,
				Body:   testCase.Log,
			}

			step.Attachments = append(
				step.Attachments, allure.Attachment{
					Name:   goTest.Name,
					Source: source,
					Type:   "plain/text",
				},
			)
		}

		switch obj := allureObj.(type) {
		case *allure.Test:
			obj.Steps = append(obj.Steps, step)
		case *allure.Step:
			obj.Steps = append(obj.Steps, step)
		default:
		}

		e.addStep(&step, tc, ch)
	}
}

func (e *exporter) defaultLabels(goTest gotest.Test, allureTest *allure.Test) {
	goTestFile, ok := e.files[goTest.Package+goTest.Name]
	if ok {
		allureTest.Labels = []allure.Label{
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

		allureTest.FullName = fmt.Sprintf("%s/%s:%s", goTestFile.PackageName, goTestFile.FileName, goTest.Name)
	}

	allureTest.Labels = append(allureTest.Labels, e.opts.allureLabels...)
}

func (*exporter) convertStatus(goTest gotest.Test) string {
	var status string
	switch goTest.Status {
	case gotest.ActionSkip:
		status = allure.StatusSkip
	case gotest.ActionFail:
		if _, ok := slice.Find(
			goTest.Output, func(v string) bool {
				return strings.Contains(v, "panic")
			},
		); ok {
			status = allure.StatusBroken
			break
		}
		status = allure.StatusFail
	case gotest.ActionPass:
		status = allure.StatusPass
	default:
		status = allure.StatusBroken
	}

	return status
}
