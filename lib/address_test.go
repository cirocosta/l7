package lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerification(t *testing.T) {
	var testCases = []struct {
		input       string
		output      string
		shouldError bool
	}{
		{
			"something.com",
			"something.com:80",
			false,
		},
		{
			"https://something.com",
			"something.com:443",
			false,
		},
		{
			"127.0.0.1",
			"127.0.0.1:80",
			false,
		},
		{
			"127.0.0.1:8080",
			"127.0.0.1:8080",
			false,
		},
		{
			"http://127.0.0.1:8080",
			"127.0.0.1:8080",
			false,
		},
		{
			"http://:8080",
			"",
			true,
		},
	}

	var (
		testName string
		actual   string
		err      error
	)

	for _, tc := range testCases {
		testName = fmt.Sprintf("normalize: [%s]-->[%s]", tc.input, tc.output)
		t.Run(testName, func(t *testing.T) {
			actual, err = NormalizeAddress(tc.input)
			assert.Equal(t, tc.shouldError, err != nil)
			assert.Equal(t, tc.output, actual)
		})
	}
}
