package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/robotomize/go-allure/internal/slice"

	"github.com/robotomize/go-allure/internal/allure"
	converter2 "github.com/robotomize/go-allure/internal/converter"
	"github.com/robotomize/go-allure/internal/gointernal"
)

var (
	verboseFlag      bool
	outputDirFlag    string
	goBuildTagsFlag  string
	allureSuiteFlag  string
	allureTagsFlag   string
	allureLayersFlag string
	allureLabelsFlag string
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&verboseFlag,
		"verbose",
		"v",
		false,
		"more verbose: -v",
	)
	rootCmd.PersistentFlags().StringVarP(
		&outputDirFlag,
		"output",
		"o",
		"",
		"output path to allure reports: -o <report-path>",
	)
	rootCmd.PersistentFlags().StringVarP(
		&goBuildTagsFlag,
		"gotags",
		"",
		"",
		"pass custom build tags: --gotags integration,fixture,linux",
	)
	rootCmd.PersistentFlags().StringVarP(
		&allureSuiteFlag,
		"allure-suite",
		"",
		"",
		"add allure suite to all tests: --allure-suite MyFirstSuite",
	)
	rootCmd.PersistentFlags().StringVarP(
		&allureTagsFlag,
		"allure-tags",
		"",
		"",
		"add allure tags to all tests: --allure-tags UNIT,ACCEPTANCE",
	)
	rootCmd.PersistentFlags().StringVarP(
		&allureLayersFlag,
		"allure-layers",
		"",
		"",
		"add allure layers to all tests: --allure-layers UNIT,FUNCTIONAL",
	)
	rootCmd.PersistentFlags().StringVarP(
		&allureLabelsFlag,
		"allure-labels",
		"",
		"",
		"add allure custom labels to all tests: --allure-labels key:value,key:value1,key1:value",
	)
}

var rootCmd = &cobra.Command{
	Use:          "golurectl",
	Long:         "Convert go test output to allure reports",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		const defaultBufCap = 4096

		if outputDirFlag != "" {
			if err := mkdir(); err != nil {
				return err
			}
		}

		var opts []converter2.Option
		if verboseFlag { // nolint
			// @TODO verbose output
		}

		pwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("os.Getwd: %w", err)
		}

		buildArgs := make([]string, 0)
		if goBuildTagsFlag != "" {
			buildArgs = append(
				buildArgs,
				append(append(buildArgs, "-tags"), strings.Split(strings.TrimSpace(goBuildTagsFlag), ",")...)...,
			)
		}

		modules, err := gointernal.ReadGoModules(ctx, pwd, buildArgs...)
		if err != nil {
			return fmt.Errorf("goallure.ReadGoModules: %w", err)
		}

		goTestFiles, err := gointernal.ReadGoTestFiles(ctx, modules)
		if err != nil {
			return fmt.Errorf("goallure.ReadGoTestFiles: %w", err)
		}

		converter := converter2.New(goTestFiles, opts...)
		stdoutOutputBuffer := bytes.NewBuffer(make([]byte, 0, defaultBufCap))
		jsonInputScanner := bufio.NewScanner(os.Stdin)
		for jsonInputScanner.Scan() {
			select {
			case <-ctx.Done():
			default:
			}

			line := jsonInputScanner.Bytes()

			var row gointernal.GoTestLogEntry
			if err := json.Unmarshal(line, &row); err != nil {
				errorf(cmd, "json.Unmarshal: %v", err)
			}

			stdoutOutputBuffer.Write([]byte(row.Output))

			converter.Append(row)
		}

		output := converter.Output()
		labels := addFlagLabels()

		for idx := range output {
			output[idx].Labels = append(output[idx].Labels, labels...)
		}

		goTestOrigScanner := bufio.NewScanner(stdoutOutputBuffer)
		for goTestOrigScanner.Scan() {
			if _, err := cmd.OutOrStdout().Write(goTestOrigScanner.Bytes()); err != nil {
				return fmt.Errorf("command OutOrStdout.Write: %w", err)
			}

			if _, err = cmd.OutOrStdout().Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("command OutOrStdout.Write: %w", err)
			}
		}

		if _, err = cmd.OutOrStdout().Write([]byte{'\n'}); err != nil {
			return fmt.Errorf("command OutOrStdout.Write: %w", err)
		}

		var failed bool
		for _, tc := range output {
			if !failed && (tc.Status == allure.StatusFail || tc.Status == allure.StatusBroken) {
				failed = true
			}
			if err := write(tc); err != nil {
				return err
			}
		}

		if failed {
			os.Exit(1)
		}

		return nil
	},
}

func addFlagLabels() []allure.Label {
	var labels []allure.Label
	if len(allureSuiteFlag) > 0 {
		labels = append(
			labels, allure.Label{
				Name:  "suite",
				Value: strings.TrimSpace(allureSuiteFlag),
			},
		)
	}

	filterEmptyStrFn := func(v string) bool {
		return len(v) > 0
	}

	filterCustomLabelsStrFn := func(v string) bool {
		tokens := strings.Split(v, ":")
		return len(tokens) == 2 && len(tokens[0]) > 0 && len(tokens[1]) > 0
	}

	mapLabelsStrFn := func(t string) allure.Label {
		tokens := strings.Split(t, ":")

		return allure.Label{
			Name:  strings.TrimSpace(tokens[0]),
			Value: strings.TrimSpace(tokens[1]),
		}
	}

	mapCustomLabelsFunc := func(name string) func(t string) allure.Label {
		return func(t string) allure.Label {
			return allure.Label{
				Name:  name,
				Value: strings.TrimSpace(t),
			}
		}
	}

	if len(allureTagsFlag) > 0 {
		labels = append(
			labels, slice.Map(
				slice.Filter(
					strings.Split(allureTagsFlag, ","), filterEmptyStrFn,
				), mapCustomLabelsFunc("tag"),
			)...,
		)
	}

	if len(allureLayersFlag) > 0 {
		labels = append(
			labels, slice.Map(
				slice.Filter(
					strings.Split(allureLayersFlag, ","), filterEmptyStrFn,
				), mapCustomLabelsFunc("layer"),
			)...,
		)
	}

	if len(allureLabelsFlag) > 0 {
		labels = append(
			labels, slice.Map(
				slice.Filter(
					slice.Filter(
						strings.Split(allureLabelsFlag, ","), filterEmptyStrFn,
					), filterCustomLabelsStrFn,
				), mapLabelsStrFn,
			)...,
		)
	}

	return labels
}

func errorf(cmd *cobra.Command, message string, args ...any) {
	if verboseFlag {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), message, args...)
	}
}

func write(tc allure.Test) error {
	dst := os.Stdout
	if outputDirFlag != "" {
		reportFile := filepath.Join(outputDirFlag, fmt.Sprintf("%s-result.json", tc.UUID.String()))

		file, err := os.OpenFile(reportFile, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			return fmt.Errorf("os.OpenFile: %w", err)
		}

		defer file.Close()
		dst = file
	}

	if err := json.NewEncoder(dst).Encode(tc); err != nil {
		return fmt.Errorf("json.NewEncoder.Encode: %w", err)
	}

	return nil
}

func mkdir() error {
	if _, err := os.Stat(outputDirFlag); os.IsNotExist(err) {
		if err = os.MkdirAll(outputDirFlag, os.ModePerm); err != nil {
			return fmt.Errorf("os.MkdirAll: %w", err)
		}
	}

	return nil
}
