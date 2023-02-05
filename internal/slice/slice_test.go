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

				converted := Map(tc.input, tc.f)
				if !reflect.DeepEqual(converted, tc.expected) {
					t.Errorf("got: %v, want: %v", converted, tc.expected)
				}
			},
		)
	}
}
