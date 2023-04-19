package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/robotomize/go-allure/internal/fs"
	"github.com/spf13/cobra"

	"github.com/robotomize/go-allure/internal/allure"
	"github.com/robotomize/go-allure/internal/exporter"
	"github.com/robotomize/go-allure/internal/golist"
	"github.com/robotomize/go-allure/internal/gotest"
	"github.com/robotomize/go-allure/internal/parser"
	"github.com/robotomize/go-allure/internal/slice"
)

var (
	verboseFlag           bool
	outputDirFlag         string
	forwardGoTestExitCode bool
	forwardGoTestLog      bool
	goBuildTagsFlag       string
	allureSuiteFlag       string
	allureTagsFlag        string
	allureLayersFlag      string
	allureLabelsFlag      string
	allureAttachmentForce bool
	silentOutput          bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&verboseFlag,
		"verbose",
		"v",
		false,
		"verbose",
	)
	rootCmd.PersistentFlags().StringVarP(
		&outputDirFlag,
		"output",
		"o",
		"",
		"output path to allure reports: -o <report-path>",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&forwardGoTestExitCode,
		"forward-exit",
		"e",
		false,
		"forward the origin go test exit code",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&forwardGoTestLog,
		"forward-log",
		"l",
		false,
		"output the origin go test",
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
	rootCmd.PersistentFlags().BoolVarP(
		&allureAttachmentForce,
		"attachment-force",
		"a",
		false,
		"create attachments for passed tests",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&silentOutput,
		"silent",
		"s",
		false,
		"silent allure report output(JSON)",
	)
}

var rootCmd = &cobra.Command{
	Use:          "golurectl",
	Long:         "Export go test output to allure reports",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		opts := []exporter.Option{
			exporter.WithAllureLabels(processAllureLabels()...),
		}

		if allureAttachmentForce {
			opts = append(opts, exporter.WithForceAttachment())
		}

		pwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("os.Getwd: %w", err)
		}

		var buildArgs []string
		if goBuildTagsFlag != "" {
			buildArgs = append([]string{"-tags"}, strings.Split(strings.TrimSpace(goBuildTagsFlag), ",")...)
		}

		pkgReader := gotest.NewReader(os.Stdin)
		goParser := parser.New(golist.NewRetriever(fs.New(pwd), buildArgs...))
		allureExporter := exporter.New(goParser, pkgReader, opts...)
		if err := allureExporter.Read(ctx); err != nil {
			return fmt.Errorf("exporter Read: %w", err)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTrying to generate an allure report\n")
		allureReport, err := allureExporter.Export()
		if err != nil {
			return fmt.Errorf("exporter Export: %w", err)
		}

		if verboseFlag && allureReport.Err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Read go test output log: %s", allureReport.Err.Error())
		}

		if forwardGoTestLog {
			if _, err := io.Copy(cmd.OutOrStdout(), allureReport.OutputLog); err != nil {
				return fmt.Errorf("io.сopy: %w", err)
			}
		}

		var outOpts []exporter.WriterOption
		if outputDirFlag != "" {
			outOpts = append(outOpts, exporter.WithOutputPth(outputDirFlag))
		}

		outputWriter := io.Writer(os.Stdout)
		if silentOutput {
			outputWriter = io.Discard
		}

		writer := exporter.NewWriter(outputWriter, outOpts...)

		if len(outputDirFlag) > 0 && len(allureReport.Tests) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Write report files\n")
		}

		if err := writer.WriteReport(ctx, allureReport.Tests); err != nil {
			return fmt.Errorf("exporter.NewWriter WriteReport: %w", err)
		}

		if len(outputDirFlag) > 0 && len(allureReport.Attachments) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Write attachments\n")
		}

		if err := writer.WriteAttachments(ctx, allureReport.Attachments); err != nil {
			return fmt.Errorf("exporter.NewWriter WriteAttachments: %w", err)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversion completed successfully\n")

		if forwardGoTestExitCode {
			var failed bool
			for _, tc := range allureReport.Tests {
				if !failed && (tc.Status == allure.StatusFail || tc.Status == allure.StatusBroken) {
					failed = true
				}
			}

			if failed {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "One or more go tests failed. exiting with error 1\n")
				os.Exit(1)
			}
		}

		return nil
	},
}

func processAllureLabels() []allure.Label {
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
