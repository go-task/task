package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionTxt(t *testing.T) {
	// Check that the version.txt is a valid semver version.
	require.NotEmpty(t, GetVersion(), "version.txt is not semver compliant")
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		version string
		commit  string
		dirty   bool
		want    string
	}{
		{"1.2.3", "", false, "1.2.3"},
		{"1.2.3", "", true, "1.3.0"},
		{"1.2.3", "abcdefg", false, "1.3.0"},
		{"1.2.3", "abcdefg", true, "1.3.0"},
	}

	for _, tt := range tests {
		version = tt.version
		commit = tt.commit
		dirty = tt.dirty
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, GetVersion())
		})
	}
}

func TestGetVersionWithBuildInfo(t *testing.T) {
	tests := []struct {
		version string
		commit  string
		dirty   bool
		want    string
	}{
		{"1.2.3", "", false, "1.2.3"},
		{"1.2.3", "", true, "1.3.0+dirty"},
		{"1.2.3", "abcdefg", false, "1.3.0+abcdefg"},
		{"1.2.3", "abcdefg", true, "1.3.0+abcdefg.dirty"},
	}

	for _, tt := range tests {
		version = tt.version
		commit = tt.commit
		dirty = tt.dirty
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, GetVersionWithBuildInfo())
		})
	}
}
