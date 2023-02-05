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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, _ := Marshal(tc.input)
			if len(result) != tc.expected {
				t.Errorf("got: %d, want: %d", len(result), tc.expected)
			}
		})
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
		t.Run(tc.name, func(t *testing.T) {
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
		})
	}
}
