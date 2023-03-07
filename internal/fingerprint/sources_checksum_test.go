package fingerprint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeFilename(t *testing.T) {
	tests := []struct {
		In, Out string
	}{
		{"foobarbaz", "foobarbaz"},
		{"foo/bar/baz", "foo-bar-baz"},
		{"foo@bar/baz", "foo-bar-baz"},
		{"foo1bar2baz3", "foo1bar2baz3"},
	}
	for _, test := range tests {
		assert.Equal(t, test.Out, normalizeFilename(test.In))
	}
}
