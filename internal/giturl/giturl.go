// Package giturl parses Git URLs.
//
// These URLs include standard RFC 3986 URLs as well as special formats that
// are specific to Git. Examples are provided in the Git documentation at
// https://mirrors.edge.kernel.org/pub/software/scm/git/docs/git-clone.html
package giturl

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// scpURLMaxLen is max length of the SCP URL to prevent reDOS attacks.
const scpURLMaxLen = 2048

var (
	// scpSyntax matches the SCP-like addresses used by Git to access repositories by SSH.
	scpSyntax = regexp.MustCompile(`^([a-zA-Z0-9-._~]+@)?([a-zA-Z0-9._-]+):([a-zA-Z0-9./._-]+)(?:\?||$)(.*)$`)

	// transports is a set of known Git URL schemes.
	transports = map[string]struct{}{
		"ssh":     {},
		"git":     {},
		"git+ssh": {},
		"http":    {},
		"https":   {},
		"ftp":     {},
		"ftps":    {},
		"rsync":   {},
		"file":    {},
	}
)

// parser converts a string into a URL.
type parser func(string) (*url.URL, error)

// Parse parses rawURL into a URL structure. Parse first attempts to find a standard URL
// with a valid Git transport as its scheme. If that cannot be found, it then attempts=
// to find a SCP-like URL. And if that cannot be found, it assumes rawURL is a local path.
// If none of these rules apply, Parse returns an error.
func Parse(rawURL string) (*url.URL, error) {
	parsers := []parser{
		parseTransport,
		parseSCP,
		parseLocal,
	}

	// Apply each parser in turn; if the parser succeeds, accept its result and return.
	var err error
	for _, p := range parsers {
		var u *url.URL
		u, err = p(rawURL)
		if err == nil {
			return u, nil
		}
	}

	// It's unlikely that none of the parsers will succeed, since
	// ParseLocal is very forgiving.
	return nil, fmt.Errorf("failed to parse %q: %w", rawURL, err)
}

// parseTransport parses rawURL into a URL object. Unless the URL's scheme is a known Git transport,
// parseTransport returns an error.
func parseTransport(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if _, ok := transports[u.Scheme]; !ok {
		return nil, fmt.Errorf("scheme %q is not a valid transport", u.Scheme)
	}
	return u, nil
}

// parseSCP parses rawURL into a URL object. The rawURL must be
// an SCP-like URL, otherwise parseSCP returns an error.
func parseSCP(rawURL string) (*url.URL, error) {
	if len(rawURL) > scpURLMaxLen {
		return nil, fmt.Errorf("URL too long: %q", rawURL)
	}
	match := scpSyntax.FindAllStringSubmatch(rawURL, -1)
	if len(match) == 0 {
		return nil, fmt.Errorf("no scp URL found in %q", rawURL)
	}
	m := match[0]
	user := strings.TrimRight(m[1], "@")
	var userinfo *url.Userinfo
	if user != "" {
		userinfo = url.User(user)
	}
	rawQuery := ""
	if len(m) > 3 {
		rawQuery = m[4]
	}
	return &url.URL{
		Scheme:   "ssh",
		User:     userinfo,
		Host:     m[2],
		Path:     m[3],
		RawQuery: rawQuery,
	}, nil
}

// parseLocal parses rawURL into a URL object with a "file" scheme.
// This will effectively never return an error.
func parseLocal(rawURL string) (*url.URL, error) {
	return &url.URL{
		Scheme: "file",
		Host:   "",
		Path:   rawURL,
	}, nil
}
