package slice

import (
	"reflect"
	"testing"
)

func TestMap(t *testing.T) {
	t.Parallel()

	mapperFunc := func(a uint32) uint64 {
		return uint64(a)
	}

	testCases := []struct {
		name     string
		f        func(a uint32) uint64
		input    []uint32
		expected []uint64
	}{
		{
			name:     "test_ok",
			f:        mapperFunc,
			input:    []uint32{1, 2, 3},
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "test_func_nil",
			input:    []uint32{1, 2, 3},
			expected: []uint64{},
		},
		{
			name:     "test_nil_input",
			f:        mapperFunc,
			expected: []uint64{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()
				t.Run(
					"test", func(t *testing.T) {
						t.Parallel()
						t.Run(
							"test1", func(t *testing.T) {
								t.Parallel()
								t.Error()
							},
						)
					},
				)
				converted := Map(tc.input, tc.f)
				if !reflect.DeepEqual(converted, tc.expected) {
					t.Errorf("got: %v, want: %v", converted, tc.expected)
				}
			},
		)
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []int
		fn       func(int) bool
		expected []int
	}{
		{
			name:     "test_filtered",
			input:    []int{1, 2, 3, 4},
			fn:       func(i int) bool { return i > 2 },
			expected: []int{3, 4},
		},
		{
			name:     "test_filtered_empty",
			input:    []int{1, 2, 3, 4},
			fn:       func(i int) bool { return i > 6 },
			expected: []int{},
		},
		{
			name:     "test_nil_input",
			input:    nil,
			fn:       func(i int) bool { return i > 6 },
			expected: []int{},
		},
		{
			name:     "test_empty_input",
			input:    nil,
			fn:       func(i int) bool { return i > 6 },
			expected: []int{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()

				output := Filter(tc.input, tc.fn)
				if !reflect.DeepEqual(output, tc.expected) {
					t.Errorf("got: %v, want: %v", output, tc.expected)
				}
			},
		)
	}
}

func TestFind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         []int
		fn            func(int) bool
		expectedValue int
		expectedExist bool
	}{
		{
			name:          "test_found",
			input:         []int{1, 2, 3, 4},
			fn:            func(i int) bool { return i == 2 },
			expectedValue: 2,
			expectedExist: true,
		},
		{
			name:          "test_not_found",
			input:         []int{1, 2, 3, 4},
			fn:            func(i int) bool { return i == 5 },
			expectedValue: 0,
			expectedExist: false,
		},
		{
			name:          "test_nil_input",
			input:         nil,
			fn:            func(i int) bool { return i > 6 },
			expectedValue: 0,
			expectedExist: false,
		},
		{
			name:          "test_empty_input",
			input:         nil,
			fn:            func(i int) bool { return i > 6 },
			expectedValue: 0,
			expectedExist: false,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()

				output, ok := Find(tc.input, tc.fn)
				if !reflect.DeepEqual(ok, tc.expectedExist) {
					t.Errorf("got: %v, want: %v", output, tc.expectedExist)
				}
				if !reflect.DeepEqual(output, tc.expectedValue) {
					t.Errorf("got: %v, want: %v", output, tc.expectedValue)
				}
			},
		)
	}
}

func TestFlat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    [][]int
		expected []int
	}{
		{
			name:     "test_rows_0",
			input:    [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
			expected: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:     "test_rows_1",
			input:    [][]int{{}, {}, {1, 2, 3}},
			expected: []int{1, 2, 3},
		},
		{
			name:     "test_rows_2",
			input:    [][]int{{1, 2, 3}, {}, {4, 5, 6}},
			expected: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:     "test_input_empty",
			input:    [][]int{},
			expected: []int{},
		},
		{
			name:     "test_input_nil",
			input:    nil,
			expected: []int{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(
			tc.name, func(t *testing.T) {
				t.Parallel()
				output := Flat(tc.input)
				if !reflect.DeepEqual(output, tc.expected) {
					t.Errorf("Flat(%v) = %v, expected %v", tc.input, output, tc.expected)
				}
			},
		)
	}
}
