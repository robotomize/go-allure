//go:build !all && integration
// +build !all,integration

package tests

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/robotomize/go-allure/internal/allure"
	"github.com/robotomize/go-allure/internal/exporter"
)

//go:embed fixtures/test_sample.txt
var testSample []byte

func TestConv(t *testing.T) {
	t.Parallel()

	// TestUnmarshal - fail
	// TestMarshal - pass
	// TestConv - pass
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	converter := exporter.New(pwd, bytes.NewReader(testSample))

	output, err := converter.Export(context.Background())
	if err != nil {
		t.Fatalf("converter Export: %v", err)
	}

	for _, tc := range output.Tests {
		switch tc.Name {
		case "TestMarshal":
			if diff := cmp.Diff(allure.StatusPass, tc.Status); diff != "" {
				t.Errorf("bad message (+got, -want): %s", diff)
			}
			// t.Errorf("error")
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
		case "TestConv":
			if diff := cmp.Diff(allure.StatusPass, tc.Status); diff != "" {
				t.Errorf("bad message (+got, -want): %s", diff)
			}
		default:
		}
	}
}
