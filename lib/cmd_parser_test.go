package lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualSeparatedToMap(t *testing.T) {
	var testCases = []struct {
		input       []string
		output      map[string][]string
		shouldError bool
	}{
		{
			[]string{},
			map[string][]string{},
			false,
		},
		{
			[]string{""},
			map[string][]string{},
			true,
		},
		{
			[]string{"abc"},
			map[string][]string{},
			true,
		},
		{
			[]string{"foo=bar"},
			map[string][]string{"foo": []string{"bar"}},
			false,
		},
		{
			[]string{"label=foo=bar"},
			map[string][]string{"label": []string{"foo=bar"}},
			false,
		},
		{
			[]string{"label=foo=bar", "label=caz=baz"},
			map[string][]string{"label": []string{"foo=bar", "caz=baz"}},
			false,
		},
	}

	var (
		testName string
		actual   map[string][]string
		err      error
	)

	for _, tc := range testCases {
		testName = fmt.Sprintf("%v-->%v", tc.input, tc.output)
		t.Run(testName, func(t *testing.T) {
			actual, err = EqualSeparatedToMap(tc.input)
			assert.Equal(t, tc.shouldError, err != nil)
			assert.Equal(t, tc.output, actual)
		})
	}
}
