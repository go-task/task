package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input        string
		ExpectedOS   string
		ExpectedArch string
		Error        string
	}{
		{Input: "windows", ExpectedOS: "windows", ExpectedArch: ""},
		{Input: "linux", ExpectedOS: "linux", ExpectedArch: ""},
		{Input: "darwin", ExpectedOS: "darwin", ExpectedArch: ""},

		{Input: "386", ExpectedOS: "", ExpectedArch: "386"},
		{Input: "amd64", ExpectedOS: "", ExpectedArch: "amd64"},
		{Input: "arm64", ExpectedOS: "", ExpectedArch: "arm64"},

		{Input: "windows/386", ExpectedOS: "windows", ExpectedArch: "386"},
		{Input: "windows/amd64", ExpectedOS: "windows", ExpectedArch: "amd64"},
		{Input: "windows/arm64", ExpectedOS: "windows", ExpectedArch: "arm64"},

		{Input: "invalid", Error: `invalid platform "invalid"`},
		{Input: "invalid/invalid", Error: `invalid platform "invalid/invalid"`},
		{Input: "windows/invalid", Error: `invalid platform "windows/invalid"`},
		{Input: "invalid/amd64", Error: `invalid platform "invalid/amd64"`},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			t.Parallel()

			var p Platform
			err := p.parsePlatform(test.Input)

			if test.Error != "" {
				require.Error(t, err)
				assert.Equal(t, test.Error, err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.ExpectedOS, p.OS)
				assert.Equal(t, test.ExpectedArch, p.Arch)
			}
		})
	}
}
