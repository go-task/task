// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

func charClass(s string) (string, error) {
	if strings.HasPrefix(s, "[[.") || strings.HasPrefix(s, "[[=") {
		return "", fmt.Errorf("collating features not available")
	}
	if !strings.HasPrefix(s, "[[:") {
		return "", nil
	}
	name := s[3:]
	end := strings.Index(name, ":]]")
	if end < 0 {
		return "", fmt.Errorf("[[: was not matched with a closing :]]")
	}
	name = name[:end]
	switch name {
	case "alnum", "alpha", "ascii", "blank", "cntrl", "digit", "graph",
		"lower", "print", "punct", "space", "upper", "word", "xdigit":
	default:
		return "", fmt.Errorf("invalid character class: %q", name)
	}
	return s[:len(name)+6], nil
}

// TranslatePattern turns a shell wildcard pattern into a regular expression
// that can be used with regexp.Compile. It will return an error if the input
// pattern was incorrect. Otherwise, the returned expression can be passed to
// regexp.MustCompile.
//
// For example, TranslatePattern(`foo*bar?`, true) returns `foo.*bar.`.
//
// Note that this function (and QuotePattern) should not be directly used with
// file paths if Windows is supported, as the path separator on that platform is
// the same character as the escaping character for shell patterns.
func TranslatePattern(pattern string, greedy bool) (string, error) {
	any := false
loop:
	for _, r := range pattern {
		switch r {
		// including those that need escaping since they are
		// special chars in regexes
		case '*', '?', '[', '\\', '.', '+', '(', ')', '|',
			']', '{', '}', '^', '$':
			any = true
			break loop
		}
	}
	if !any { // short-cut without a string copy
		return pattern, nil
	}
	var buf bytes.Buffer
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '*':
			buf.WriteString(".*")
			if !greedy {
				buf.WriteByte('?')
			}
		case '?':
			buf.WriteString(".")
		case '\\':
			if i++; i >= len(pattern) {
				return "", fmt.Errorf(`\ at end of pattern`)
			}
			buf.WriteString(regexp.QuoteMeta(string(pattern[i])))
		case '[':
			name, err := charClass(pattern[i:])
			if err != nil {
				return "", err
			}
			if name != "" {
				buf.WriteString(name)
				i += len(name) - 1
				break
			}
			buf.WriteByte(c)
			if i++; i >= len(pattern) {
				return "", fmt.Errorf("[ was not matched with a closing ]")
			}
			switch c = pattern[i]; c {
			case '!', '^':
				buf.WriteByte('^')
				i++
				c = pattern[i]
			}
			buf.WriteByte(c)
			last := c
			rangeStart := byte(0)
			for {
				if i++; i >= len(pattern) {
					return "", fmt.Errorf("[ was not matched with a closing ]")
				}
				last, c = c, pattern[i]
				buf.WriteByte(c)
				if c == ']' {
					break
				}
				if rangeStart != 0 && rangeStart > c {
					return "", fmt.Errorf("invalid range: %c-%c", rangeStart, c)
				}
				if c == '-' {
					rangeStart = last
				} else {
					rangeStart = 0
				}
			}
		default:
			buf.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return buf.String(), nil
}

// HasPattern returns whether a string contains any unescaped wildcard
// characters: '*', '?', or '['. When the function returns false, the given
// pattern can only match at most one string.
//
// For example, HasPattern(`foo\*bar`) returns false, but HasPattern(`foo*bar`)
// returns true.
//
// This can be useful to avoid extra work, like TranslatePattern. Note that this
// function cannot be used to avoid QuotePattern, as backslashes are quoted by
// that function but ignored here.
func HasPattern(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			i++
		case '*', '?', '[':
			return true
		}
	}
	return false
}

// QuotePattern returns a string that quotes all special characters in the given
// wildcard pattern. The returned string is a pattern that matches the literal
// string.
//
// For example, QuotePattern(`foo*bar?`) returns `foo\*bar\?`.
func QuotePattern(pattern string) string {
	any := false
loop:
	for _, r := range pattern {
		switch r {
		case '*', '?', '[', '\\':
			any = true
			break loop
		}
	}
	if !any { // short-cut without a string copy
		return pattern
	}
	var buf bytes.Buffer
	for _, r := range pattern {
		switch r {
		case '*', '?', '[', '\\':
			buf.WriteByte('\\')
		}
		buf.WriteRune(r)
	}
	return buf.String()
}
