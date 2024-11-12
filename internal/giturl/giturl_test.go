package giturl

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		wantURL *url.URL
		wantStr string
		wantErr bool
	}{
		{
			"user@host.xz:path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				User:   url.User("user"),
				Host:   "host.xz",
				Path:   "path/to/repo.git/",
			},
			"ssh://user@host.xz/path/to/repo.git/",
			false,
		},
		{
			"host.xz:path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				Host:   "host.xz",
				Path:   "path/to/repo.git/",
			},
			"ssh://host.xz/path/to/repo.git/",
			false,
		},
		{
			"host.xz:/path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"ssh://host.xz/path/to/repo.git/",
			false,
		},
		{
			"host.xz:path/to/repo-with_specials.git/",
			&url.URL{
				Scheme: "ssh",
				Host:   "host.xz",
				Path:   "path/to/repo-with_specials.git/",
			},
			"ssh://host.xz/path/to/repo-with_specials.git/",
			false,
		},
		{
			"git://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "git",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"git://host.xz/path/to/repo.git/",
			false,
		},
		{
			"git://host.xz:1234/path/to/repo.git/",
			&url.URL{
				Scheme: "git",
				Host:   "host.xz:1234",
				Path:   "/path/to/repo.git/",
			},
			"git://host.xz:1234/path/to/repo.git/",
			false,
		},
		{
			"http://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "http",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"http://host.xz/path/to/repo.git/",
			false,
		},
		{
			"http://host.xz:1234/path/to/repo.git/",
			&url.URL{
				Scheme: "http",
				Host:   "host.xz:1234",
				Path:   "/path/to/repo.git/",
			},
			"http://host.xz:1234/path/to/repo.git/",
			false,
		},
		{
			"https://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "https",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"https://host.xz/path/to/repo.git/",
			false,
		},
		{
			"https://host.xz:1234/path/to/repo.git/",
			&url.URL{
				Scheme: "https",
				Host:   "host.xz:1234",
				Path:   "/path/to/repo.git/",
			},
			"https://host.xz:1234/path/to/repo.git/",
			false,
		},
		{
			"ftp://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "ftp",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"ftp://host.xz/path/to/repo.git/",
			false,
		},
		{
			"ftp://host.xz:1234/path/to/repo.git/",
			&url.URL{
				Scheme: "ftp",
				Host:   "host.xz:1234",
				Path:   "/path/to/repo.git/",
			},
			"ftp://host.xz:1234/path/to/repo.git/",
			false,
		},
		{
			"ftps://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "ftps",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"ftps://host.xz/path/to/repo.git/",
			false,
		},
		{
			"ftps://host.xz:1234/path/to/repo.git/",
			&url.URL{
				Scheme: "ftps",
				Host:   "host.xz:1234",
				Path:   "/path/to/repo.git/",
			},
			"ftps://host.xz:1234/path/to/repo.git/",
			false,
		},
		{
			"rsync://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "rsync",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"rsync://host.xz/path/to/repo.git/",
			false,
		},
		{
			"ssh://user@host.xz:1234/path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				User:   url.User("user"),
				Host:   "host.xz:1234",
				Path:   "/path/to/repo.git/",
			},
			"ssh://user@host.xz:1234/path/to/repo.git/",
			false,
		},
		{
			"ssh://host.xz:1234/path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				Host:   "host.xz:1234",
				Path:   "/path/to/repo.git/",
			},
			"ssh://host.xz:1234/path/to/repo.git/",
			false,
		},
		{
			"ssh://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"ssh://host.xz/path/to/repo.git/",
			false,
		},
		{
			"git+ssh://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "git+ssh",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			"git+ssh://host.xz/path/to/repo.git/",
			false,
		},
		{
			"/path/to/repo.git/",
			&url.URL{
				Scheme: "file",
				Path:   "/path/to/repo.git/",
			},
			"file:///path/to/repo.git/",
			false,
		},
		{
			"file:///path/to/repo.git/",
			&url.URL{
				Scheme: "file",
				Path:   "/path/to/repo.git/",
			},
			"file:///path/to/repo.git/",
			false,
		},
		{
			"https://host.xz/organization/repo.git?ref=",
			&url.URL{
				Scheme:   "https",
				Host:     "host.xz",
				Path:     "/organization/repo.git",
				RawQuery: "ref=",
			},
			"https://host.xz/organization/repo.git?ref=",
			false,
		},
		{
			"https://host.xz/organization/repo.git?ref=test",
			&url.URL{
				Scheme:   "https",
				Host:     "host.xz",
				Path:     "/organization/repo.git",
				RawQuery: "ref=test",
			},
			"https://host.xz/organization/repo.git?ref=test",
			false,
		},
		{
			"https://host.xz/organization/repo.git?ref=feature/test",
			&url.URL{
				Scheme:   "https",
				Host:     "host.xz",
				Path:     "/organization/repo.git",
				RawQuery: "ref=feature/test",
			},
			"https://host.xz/organization/repo.git?ref=feature/test",
			false,
		},
		{
			"git@host.xz:organization/repo.git?ref=test",
			&url.URL{
				Scheme:   "ssh",
				User:     url.User("git"),
				Host:     "host.xz",
				Path:     "organization/repo.git",
				RawQuery: "ref=test",
			},
			"ssh://git@host.xz/organization/repo.git?ref=test",
			false,
		},
		{
			"git@host.xz:organization/repo.git?ref=feature/test",
			&url.URL{
				Scheme:   "ssh",
				User:     url.User("git"),
				Host:     "host.xz",
				Path:     "organization/repo.git",
				RawQuery: "ref=feature/test",
			},
			"ssh://git@host.xz/organization/repo.git?ref=feature/test",
			false,
		},
		{
			"https://user:password@host.xz/organization/repo.git/",
			&url.URL{
				Scheme: "https",
				User:   url.UserPassword("user", "password"),
				Host:   "host.xz",
				Path:   "/organization/repo.git/",
			},
			"https://user:password@host.xz/organization/repo.git/",
			false,
		},
		{
			"https://user:password@host.xz/organization/repo.git?ref=test",
			&url.URL{
				Scheme:   "https",
				User:     url.UserPassword("user", "password"),
				Host:     "host.xz",
				Path:     "/organization/repo.git",
				RawQuery: "ref=test",
			},
			"https://user:password@host.xz/organization/repo.git?ref=test",
			false,
		},
		{
			"https://user:password@host.xz/organization/repo.git?ref=feature/test",
			&url.URL{
				Scheme:   "https",
				User:     url.UserPassword("user", "password"),
				Host:     "host.xz",
				Path:     "/organization/repo.git",
				RawQuery: "ref=feature/test",
			},
			"https://user:password@host.xz/organization/repo.git?ref=feature/test",
			false,
		},
		{
			"user-1234@host.xz:path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				User:   url.User("user-1234"),
				Host:   "host.xz",
				Path:   "path/to/repo.git/",
			},
			"ssh://user-1234@host.xz/path/to/repo.git/",
			false,
		},
	}

	for _, tt := range tests {

		got, err := Parse(tt.in)
		if tt.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got)
			assert.Equal(t, tt.wantStr, got.String())
		}
	}
}

func TestParseSCP(t *testing.T) {
	tests := []struct {
		in      string
		wantURL *url.URL
		wantErr bool
	}{
		{
			"user@host.xz:path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				User:   url.User("user"),
				Host:   "host.xz",
				Path:   "path/to/repo.git/",
			},
			false,
		},
		{
			"host.xz:path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				Host:   "host.xz",
				Path:   "path/to/repo.git/",
			},
			false,
		},
		{
			"host.xz:/path/to/repo.git/",
			&url.URL{
				Scheme: "ssh",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			}, false,
		},
		{
			"invalid-scp-url",
			nil,
			true,
		},
		{
			"https://example.com/" + strings.Repeat("a", 4049),
			nil,
			true,
		},
	}

	for _, tt := range tests {
		got, err := parseSCP(tt.in)
		if tt.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got)
		}
	}
}

func TestParseTransport(t *testing.T) {
	tests := []struct {
		in      string
		wantURL *url.URL
		wantErr bool
	}{
		{
			"git://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "git",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			false,
		},
		{
			"http://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "http",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			false,
		},
		{
			"https://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "https",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			false,
		},
		{
			"ftp://host.xz/path/to/repo.git/",
			&url.URL{
				Scheme: "ftp",
				Host:   "host.xz",
				Path:   "/path/to/repo.git/",
			},
			false,
		},
		{
			"invalid://host.xz/path/to/repo.git/",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		got, err := parseTransport(tt.in)
		if tt.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got)
		}
	}
}

func TestParseLocal(t *testing.T) {
	rawURL := "/path/to/repo.git/"

	u, err := parseLocal(rawURL)

	require.NoError(t, err)
	assert.Equal(t, "file", u.Scheme)
	assert.Empty(t, u.Host)
	assert.Equal(t, "/path/to/repo.git/", u.Path)
}
