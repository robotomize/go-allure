# ⚡ Go-allure

[![Build status](https://github.com/robotomize/go-allure/actions/workflows/release.yml/badge.svg)](https://github.com/robotomize/go-allure/actions)
[![GitHub license](https://img.shields.io/github/license/robotomize/go-allure.svg)](https://github.com/robotomize/go-allure/blob/main/LICENSE)

A command line utility for converting the output of Go tests into [allure reports](https://github.com/allure-framework).

## Install

### Go

```sh
go install github.com/robotomize/go-allure/cmd/golurectl@latest
```

### Docker

```sh
docker pull robotomize/golurectl:latest
```

## Usage

```sh
Export go test output to allure reports

Usage:
  golurectl [flags]
  golurectl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     actual version

Flags:
      --allure-labels string   add allure custom labels to all tests: --allure-labels key:value,key:value1,key1:value
      --allure-layers string   add allure layers to all tests: --allure-layers UNIT,FUNCTIONAL
      --allure-suite string    add allure suite to all tests: --allure-suite MyFirstSuite
      --allure-tags string     add allure tags to all tests: --allure-tags UNIT,ACCEPTANCE
  -a, --attachment-force       add a log of pass and failed tests to the attachments
  -e, --forward-exit           forward the origin go test exit code
  -l, --forward-log            output the origin go test
      --gotags string          pass custom build tags: --gotags integration,fixture,linux
  -h, --help                   help for golurectl
  -o, --output string          output path to allure reports: -o <report-path>
  -s, --silent                 silent allure report output
  -v, --verbose                verbose

Use "golurectl [command] --help" for more information about a command.
```

Example output to stdout

```shell
go test -json./...|golurectl
```

Example output to reports= dir

```shell
go test -json ./...|golurectl -o reports-dir
```

Example output to report dir with flags

```shell
go test -json -cover  ./...|golurectl -l -e -o reports-dir --gotags integration --allure-suite MySuite --allure-labels epic:my_epic,custom:value --allure-tags UNIT,GO-ALLURE --allure-layers UNIT
```

## Examples

We have the following set of tests along with subtests

```go
//go:build !all && fixtures
// +build !all,fixtures

package tests

import (
	"reflect"
	"testing"
)

func TestMarshal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    simpleStruct
		expected int
	}{
		{
			name: "test_ok",
			input: simpleStruct{
				Name:     "hello",
				LastName: "world",
			},
			expected: 4 + len("hello") + len("world") + 4,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()
				result, _ := Marshal(tc.input)
				if len(result) != tc.expected {
					t.Errorf("got: %d, want: %d", len(result), tc.expected)
				}
			}
		)
	}
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    simpleStruct
		expected int
	}{
		{
			name: "test_ok",
			input: simpleStruct{
				Name:     "hello",
				LastName: "world",
			},
			expected: 4 + len("hello") + len("world") + 4,
		},
		{
			name: "test_failed",
			input: simpleStruct{
				Name:     "hello",
				LastName: "world",
			},
			expected: 4 + len("hello") + len("world") + 6,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()
				result, _ := Marshal(tc.input)
				if len(result) != tc.expected {
					t.Errorf("got: %d, want: %d", len(result), tc.expected)
				}
				var sStruct simpleStruct
				if err := Unmarshal(result, &sStruct); err != nil {
					t.Errorf("Unmarshal: %v", err)
				}
				if !reflect.DeepEqual(sStruct, tc.input) {
					t.Errorf("got: %v, want: %v", false, true)
				}
			}
		)
	}
}


```

We can feed the go test output into golurectl.
golurectl will display the results of the go test and generate a report in allure.

```shell
go test -tags=fixtures -race -json ./...|golurectl --gotags fixtures
```

```shell
{
  "uuid": "96b84a3b-2aaa-4b75-82f1-63ff3cb135db",
  "testCaseId": "979afbf07b7b262f3bfb077b2f06f8de",
  "historyId": "979afbf07b7b262f3bfb077b2f06f8de",
  "name": "TestMarshal",
  "description": "",
  "status": "passed",
  "stage": "finished",
  "steps": [
    {
      "name": "TestMarshal/test_ok",
      "status": "passed",
      "stage": "finished",
      "steps": [],
      "attachments": [],
      "parameters": [],
      "start": 1676382158,
      "stop": 1676382158
    }
  ],
  "start": 1676382158,
  "stop": 1676382158,
  "fullName": "github.com/robotomize/go-allure/tests/encoding_binary_test.go:TestMarshal",
  "parameters": [],
  "labels": [
    {
      "name": "package",
      "value": "github.com/robotomize/go-allure/tests"
    },
    {
      "name": "testClass",
      "value": "github.com/robotomize/go-allure/tests/encoding_binary_test.go"
    },
    {
      "name": "testMethod",
      "value": "TestMarshal"
    },
    {
      "name": "language",
      "value": "golang"
    },
    {
      "name": "go-version",
      "value": "1.19"
    },
    {
      "name": "host",
      "value": "popos"
    }
  ],
  "attachments": []
}

{
  "uuid": "587d51b4-3947-46cf-8d6a-3ea0376dcdd8",
  "testCaseId": "d9e4658d46475d46c45536dc9cc4bdb7",
  "historyId": "d9e4658d46475d46c45536dc9cc4bdb7",
  "name": "TestUnmarshal",
  "description": "",
  "status": "failed",
  "stage": "finished",
  "steps": [
    {
      "name": "TestUnmarshal/test_ok",
      "status": "passed",
      "stage": "finished",
      "steps": [],
      "attachments": [],
      "parameters": [],
      "start": 1676382158,
      "stop": 1676382158
    },
    {
      "name": "TestUnmarshal/test_failed",
      "status": "failed",
      "stage": "finished",
      "steps": [],
      "attachments": [],
      "parameters": [],
      "start": 1676382158,
      "stop": 1676382158
    }
  ],
  "start": 1676382158,
  "stop": 1676382158,
  "fullName": "github.com/robotomize/go-allure/tests/encoding_binary_test.go:TestUnmarshal",
  "parameters": [],
  "labels": [
    {
      "name": "package",
      "value": "github.com/robotomize/go-allure/tests"
    },
    {
      "name": "testClass",
      "value": "github.com/robotomize/go-allure/tests/encoding_binary_test.go"
    },
    {
      "name": "testMethod",
      "value": "TestUnmarshal"
    },
    {
      "name": "language",
      "value": "golang"
    },
    {
      "name": "go-version",
      "value": "1.19"
    },
    {
      "name": "host",
      "value": "popos"
    }
  ],
  "attachments": []
}
```
