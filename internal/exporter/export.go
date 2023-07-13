package exporter

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/robotomize/go-allure/internal/allure"
	"github.com/robotomize/go-allure/internal/gotest"
	"github.com/robotomize/go-allure/internal/parser"
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
	ParseFiles(ctx context.Context) ([]parser.GoTestMethod, error)
}

type AllureExporter interface {
	Read(ctx context.Context) error
	Export() (Report, error)
}

func New(fileParser FileParser, reader Reader, opts ...Option) AllureExporter {
	c := exporter{
		stdinReader: reader,
		fileParser:  fileParser,
		files:       make(map[string]parser.GoTestMethod),
	}

	for _, o := range opts {
		o(&c.opts)
	}

	return &c
}

type exporter struct {
	opts        Options
	originLog   io.Reader
	readErr     error
	fileParser  FileParser
	stdinReader Reader
	tests       []gotest.NestedTest
	files       map[string]parser.GoTestMethod
}

// Read reads the test files using the file parser, saves them in a map and reads the test output from stdi.
func (e *exporter) Read(ctx context.Context) error {
	// Parse the files using the file parser and save them in a map.
	files, err := e.fileParser.ParseFiles(ctx)
	if err != nil {
		return fmt.Errorf("go parser ParseFiles: %w", err)
	}

	for _, file := range files {
		key := file.PackageName + file.TestName
		e.files[key] = file
	}

	// Read the test output from stdin using the stdin reader and save the results in the exporter.
	set, err := e.stdinReader.ReadAll(ctx)
	if err != nil {
		return fmt.Errorf("stdin reader ReadAll: %w", err)
	}

	e.tests = make([]gotest.NestedTest, len(set.Tests))
	copy(e.tests, set.Tests)

	e.readErr = set.Err
	e.originLog = set.OriginLog

	return nil
}

// Export converts Go test results to Allure test report format.
func (e *exporter) Export() (Report, error) {
	result := Report{
		Err:       e.readErr,
		OutputLog: e.originLog,
	}

	attachmentCh := make(chan Attachment)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Process attachments in the channel and update them in the Report.
	go func() {
		defer wg.Done()
		for attachment := range attachmentCh {
			result.Attachments = append(result.Attachments, attachment)
		}
	}()

	// Prepare a hash function to calculate unique IDs for Allure test cases.
	hashFn := md5.New()
	hasher := func(b []byte) []byte {
		hashFn.Reset()
		hashFn.Write(b)
		return hashFn.Sum(nil)
	}

	// Iterate through each Go test case and create an Allure test case with associated metadata and attachments.
	for _, testCase := range e.tests {
		goTest := testCase.Value

		// Generate a unique ID for the Allure test case and determine its status based on the Go test status.
		id := uuid.New().String()
		status := e.convertStatus(goTest, testCase.Log)

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

		// Add default labels to the Allure test case
		e.defaultLabels(goTest, &allureTestCase)

		goTestFile, ok := e.files[goTest.Package+goTest.Name]
		if ok {
			allureTestCase.Description = goTestFile.TestComment
			allureTestCase.FullName = fmt.Sprintf("%s/%s:%s", goTestFile.PackageName, goTestFile.FileName, goTest.Name)
		}

		// Calculate test case ID as test case full name
		testCaseID := hasher([]byte(allureTestCase.FullName))
		// Generate history ID as hash of test case ID
		historyID := hasher(testCaseID)

		allureTestCase.TestCaseID = hex.EncodeToString(testCaseID)
		allureTestCase.HistoryID = hex.EncodeToString(historyID)
		allureTestCase.Start = goTest.Start.UnixMilli()
		allureTestCase.Stop = goTest.Stop.UnixMilli()

		// Check if the Go test case has a panic or failure and add the test case log as an attachment to the Allure test case.
		// Also, add a corresponding attachment to the Allure test case to enable viewing of the test case log in the report.
		hasAttachment := e.opts.forceAttachment || goTest.Status == gotest.ActionPanic || goTest.Status == gotest.ActionFail
		if hasAttachment {
			source := fmt.Sprintf("%s-attachment.txt", uuid.New().String())
			mime := "application/json"
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
					Type:   "application/json",
				},
			)
		}

		// Add test steps to the Allure test case and add it to the Report.
		e.addStep(&allureTestCase, testCase, attachmentCh)
		result.Tests = append(result.Tests, allureTestCase)
	}

	close(attachmentCh)
	wg.Wait()

	return result, nil
}

// addStep appends Allure test steps to a given Allure object from a given list of nested Go test cases.
func (e *exporter) addStep(allureObj any, testCase gotest.NestedTest, ch chan<- Attachment) {
	// Iterate through each child test case and create an Allure step with metadata and associated attachments.
	for _, tc := range testCase.Children {
		goTest := tc.Value
		// If the Go test time values are invalid, set them to the current time.
		if goTest.Stop.Before(goTest.Start) {
			goTest.Stop = time.Now()
		}

		// Get the test case name and status and create an Allure step with it.
		name := goTest.Name
		status := e.convertStatus(goTest, tc.Log)
		if status == allure.StatusBroken {
			switch obj := allureObj.(type) {
			case *allure.Test:
				obj.Status = status
			case *allure.Step:
				obj.Status = status
			default:
			}
		}
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

		// Check if the Go test case has a panic or failure and add the test case log as an attachment to the Allure step.
		// Also, add a corresponding attachment to the Allure step to enable viewing of the test case log in the report.
		hasAttachment := e.opts.forceAttachment || goTest.Status == gotest.ActionPanic || goTest.Status == gotest.ActionFail
		if hasAttachment {
			source := fmt.Sprintf("%s-attachment.txt", uuid.New().String())
			mime := "application/json"

			// It also saves attachments from the Go test cases if they are present
			ch <- Attachment{
				Name:   goTest.Name,
				Mime:   mime,
				Source: source,
				Body:   tc.Log,
			}

			step.Attachments = append(
				step.Attachments, allure.Attachment{
					Name:   goTest.Name,
					Source: source,
					Type:   "application/json",
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
	}

	allureTest.Labels = append(allureTest.Labels, e.opts.allureLabels...)
}

func (*exporter) convertStatus(goTest gotest.Test, log []byte) string {
	var status string
	switch goTest.Status {
	case gotest.ActionSkip:
		status = allure.StatusSkip
	case gotest.ActionFail:
		if bytes.Contains(log, []byte("panic")) {
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
