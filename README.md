# ⚡ Go-allure

[![Go Report](https://goreportcard.com/badge/github.com/robotomize/go-allure)](https://goreportcard.com/report/github.com/robotomize/go-allure)
[![codebeat badge](https://codebeat.co/badges/dff72628-75b8-4809-a93d-cbcbc16e5c06)](https://codebeat.co/projects/github-com-robotomize-go-allure-main)
[![codecov](https://codecov.io/gh/robotomize/go-allure/branch/main/graph/badge.svg)](https://codecov.io/gh/robotomize/go-allure)
[![Build status](https://github.com/robotomize/go-allure/actions/workflows/release.yml/badge.svg)](https://github.com/robotomize/go-allure/actions)
[![GitHub license](https://img.shields.io/github/license/robotomize/go-allure.svg)](https://github.com/robotomize/go-allure/blob/main/LICENSE)


A command line utility for converting the output of Go tests into [allure reports](https://github.com/allure-framework).

## Demo

![demo](https://github.com/robotomize/go-allure/raw/main/_media/example_2.gif)

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
  -a, --attachment-force       create attachments for passed tests
  -e, --forward-exit           forward the origin go test exit code
  -l, --forward-log            output the origin go test
      --gotags string          pass custom build tags: --gotags integration,fixture,linux
  -h, --help                   help for golurectl
  -o, --output string          output path to allure reports: -o <report-path>
  -s, --silent                 silent allure report output(JSON)
  -v, --verbose                verbose

Use "golurectl [command] --help" for more information about a command.
```

## Getting started

To quickly see how golurectl works, you can use the following guide

```shell
go install github.com/robotomize/go-allure/cmd/golurectl@latest
cd <go-project-dir>
go test -json -cover ./...|golurectl -l -e
```

A more complex example with the generation of report files and attachments
```shell
go test -json -cover ./...|golurectl -l -e -s -a -o ~/Downloads/reports --allure-suite MySuite --allure-labels epic:my_epic,custom:value --allure-tags UNIT,GO-ALLURE --allure-layers UNIT
```
### Demo with reports
![demo](https://github.com/robotomize/go-allure/raw/main/_media/getting_started.gif)
