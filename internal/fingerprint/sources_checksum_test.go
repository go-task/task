package fingerprint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		In, Out string
	}{
		{"foobarbaz", "foobarbaz"},
		{"foo/bar/baz", "foo-bar-baz"},
		{"foo@bar/baz", "foo-bar-baz"},
		{"foo1bar2baz3", "foo1bar2baz3"},
		{"foo\\bar", "foo-bar"},
		{"foo_bar", "foo-bar"},
		{"foo[bar]baz", "foo-bar-baz"},
		{"foo^bar`baz", "foo-bar-baz"},
	}
	for _, test := range tests {
		assert.Equal(t, test.Out, normalizeFilename(test.In))
	}
}
