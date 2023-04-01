package gotest

import (
	"context"
	_ "embed"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/robotomize/go-allure/internal/slice"
)

//go:embed testdata/positive_full_marshal.txt
var positiveFullMarshal string

//go:embed testdata/negative_full_marshal.txt
var negativeFullMarshal string

//go:embed testdata/negative_unmarshal_one.txt
var negativeUnmarshalOne string

func TestReader_ReadAll(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		input      io.Reader
		expected   NestedTest
		marshalErr bool
	}{
		{
			name:  "test_full_marshal_pass",
			input: strings.NewReader(positiveFullMarshal),
			expected: NestedTest{

				Value: Test{
					Name:    "TestFilter",
					Package: "github.com/robotomize/go-allure/internal/slice",
					Status:  "pass",
				},
				Children: []NestedTest{
					{
						Value: Test{
							Name:    "TestFilter/test_filtered",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_filtered_empty",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_nil_input",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_empty_input",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
				},
			},
		},
		{
			name:  "test_full_marshal_fail",
			input: strings.NewReader(negativeFullMarshal),
			expected: NestedTest{

				Value: Test{
					Name:    "TestFilter",
					Package: "github.com/robotomize/go-allure/internal/slice",
					Status:  "fail",
				},
				Children: []NestedTest{
					{
						Value: Test{
							Name:    "TestFilter/test_filtered",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "fail",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_filtered_empty",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_nil_input",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_empty_input",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
				},
			},
		},
		{
			name:       "test_unmarshal_one_fail",
			input:      strings.NewReader(negativeUnmarshalOne),
			marshalErr: true,
			expected: NestedTest{
				Value: Test{
					Name:    "TestFilter",
					Package: "github.com/robotomize/go-allure/internal/slice",
					Status:  "fail",
				},
				Children: []NestedTest{
					{
						Value: Test{
							Name:    "TestFilter/test_filtered",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "fail",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_filtered_empty",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_nil_input",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
					{
						Value: Test{
							Name:    "TestFilter/test_empty_input",
							Package: "github.com/robotomize/go-allure/internal/slice",
							Status:  "pass",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()
				ctx := context.Background()
				reader := NewReader(tc.input)
				all, err := reader.ReadAll(ctx)
				if err != nil {
					t.Fatal(err)
				}

				if all.Err != nil && !tc.marshalErr {
					t.Errorf("got: %v, want: %v", all.Err, tc.marshalErr)
				}

				testCase := all.Tests[0]

				if diff := cmp.Diff(tc.expected.Value.Name, testCase.Value.Name); diff != "" {
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}

				if diff := cmp.Diff(tc.expected.Value.Package, testCase.Value.Package); diff != "" {
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}

				if diff := cmp.Diff(tc.expected.Value.Status, testCase.Value.Status); diff != "" {
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}

				for _, childTestCase := range tc.expected.Children {
					childTestCase1, ok := slice.Find(
						testCase.Children, func(t NestedTest) bool {
							return childTestCase.Value.Name == t.Value.Name
						},
					)
					if !ok {
						t.Errorf("got: %v,want: %v", ok, true)
						return
					}

					if diff := cmp.Diff(childTestCase.Value.Package, childTestCase1.Value.Package); diff != "" {
						t.Errorf("mismatch (-want, +got):\n%s", diff)
					}

					if diff := cmp.Diff(childTestCase.Value.Status, childTestCase1.Value.Status); diff != "" {
						t.Errorf("mismatch (-want, +got):\n%s", diff)
					}
				}

				_ = all
			},
		)
	}
}
