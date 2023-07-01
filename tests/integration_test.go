//go:build !all && integration
// +build !all,integration

package tests

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/robotomize/go-allure/internal/fs"

	"github.com/robotomize/go-allure/internal/golist"
	"github.com/robotomize/go-allure/internal/gotest"
	"github.com/robotomize/go-allure/internal/parser"

	"github.com/robotomize/go-allure/internal/allure"
	"github.com/robotomize/go-allure/internal/exporter"
)

//go:embed testdata
var ffs embed.FS

func TestExport(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// TestUnmarshal - fail
	// TestMarshal - pass
	// TestExport - pass
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	testSet, err := ffs.ReadFile("testdata/current_snapshot.txt")
	if err != nil {
		t.Fatalf("fs ReadFile: %v", err)
	}

	absPth, err := filepath.Abs(filepath.Join(pwd, "../"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	r := gotest.NewReader(bytes.NewReader(testSet))
	w := exporter.NewWriter()
	p := parser.New(golist.NewRetriever(fs.New(absPth)))

	e := exporter.New(p, r)
	if err = e.Read(ctx); err != nil {
		t.Fatalf("exporter.New Read: %v", err)
	}

	output, err := e.Export()
	if err != nil {
		t.Fatalf("converter Export: %v", err)
	}

	if err = w.WriteReport(ctx, output.Tests); err != nil {
		t.Fatalf("exporter.NewWriter WriteReport: %v", err)
	}

	for _, tc := range output.Tests {
		switch tc.Name {
		case "TestMarshal":
			if diff := cmp.Diff(allure.StatusPass, tc.Status); diff != "" {
				t.Errorf("bad message (+got, -want): %s", diff)
			}
		case "TestUnmarshal":
			if diff := cmp.Diff(allure.StatusFail, tc.Status); diff != "" {
				t.Errorf("bad message (+got, -want): %s", diff)
			}
			for _, st := range tc.Steps {
				switch st.Name {
				case "test_failed":
					if diff := cmp.Diff(allure.StatusFail, tc.Status); diff != "" {
						t.Errorf("bad message (+got, -want): %s", diff)
					}
				case "test_ok":
					if diff := cmp.Diff(allure.StatusPass, tc.Status); diff != "" {
						t.Errorf("bad message (+got, -want): %s", diff)
					}
				default:
				}
			}
		default:
		}
	}
}
