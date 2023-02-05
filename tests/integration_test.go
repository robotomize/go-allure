//go:build !all && integration
// +build !all,integration

package tests

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/robotomize/go-allure/internal/allure"
	converter2 "github.com/robotomize/go-allure/internal/converter"
	goallure "github.com/robotomize/go-allure/internal/gointernal"
)

//go:embed fixtures/test_sample.txt
var testSample []byte

func TestConv(t *testing.T) {
	t.Parallel()

	// TestUnmarshal - fail
	// TestMarshal - pass
	// TestConv - pass
	converter := converter2.New(nil)
	scanner := bufio.NewScanner(bytes.NewReader(testSample))
	for scanner.Scan() {
		line := scanner.Bytes()
		var row goallure.GoTestLogEntry
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}

		converter.Append(row)
	}

	output := converter.Output()
	for _, tc := range output {
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
