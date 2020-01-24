// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// Package pattern allows working with shell pattern matching notation, also
// known as wildcards or globbing.
//
// For reference, see
// https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html#tag_18_13.
package pattern

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Mode can be used to supply a number of options to the package's functions.
// Not all functions change their behavior with all of the options below.
type Mode uint

const (
	Shortest  Mode = 1 << iota // prefer the shortest match.
	Filenames                  // "*" and "?" don't match slashes; only "**" does
	Braces                     // support "{a,b}" and "{1..4}"
)

var numRange = regexp.MustCompile(`^([+-]?\d+)\.\.([+-]?\d+)}`)

// Regexp turns a shell pattern into a regular expression that can be used with
// regexp.Compile. It will return an error if the input pattern was incorrect.
// Otherwise, the returned expression can be passed to regexp.MustCompile.
//
// For example, Regexp(`foo*bar?`, true) returns `foo.*bar.`.
//
// Note that this function (and QuoteMeta) should not be directly used with file
// paths if Windows is supported, as the path separator on that platform is the
// same character as the escaping character for shell patterns.
func Regexp(pat string, mode Mode) (string, error) {
	any := false
noopLoop:
	for _, r := range pat {
		switch r {
		// including those that need escaping since they are
		// regular expression metacharacters
		case '*', '?', '[', '\\', '.', '+', '(', ')', '|',
			']', '{', '}', '^', '$':
			any = true
			break noopLoop
		}
	}
	if !any { // short-cut without a string copy
		return pat, nil
	}
	closingBraces := []int{}
	var buf bytes.Buffer
writeLoop:
	for i := 0; i < len(pat); i++ {
		switch c := pat[i]; c {
		case '*':
			if mode&Filenames != 0 {
				if i++; i < len(pat) && pat[i] == '*' {
					if i++; i < len(pat) && pat[i] == '/' {
						buf.WriteString("(.*/|)")
					} else {
						buf.WriteString(".*")
						i--
					}
				} else {
					buf.WriteString("[^/]*")
					i--
				}
			} else {
				buf.WriteString(".*")
			}
			if mode&Shortest != 0 {
				buf.WriteByte('?')
			}
		case '?':
			if mode&Filenames != 0 {
				buf.WriteString("[^/]")
			} else {
				buf.WriteByte('.')
			}
		case '\\':
			if i++; i >= len(pat) {
				return "", fmt.Errorf(`\ at end of pattern`)
			}
			buf.WriteString(regexp.QuoteMeta(string(pat[i])))
		case '[':
			name, err := charClass(pat[i:])
			if err != nil {
				return "", err
			}
			if name != "" {
				buf.WriteString(name)
				i += len(name) - 1
				break
			}
			if mode&Filenames != 0 {
				for _, c := range pat[i:] {
					if c == ']' {
						break
					} else if c == '/' {
						buf.WriteString("\\[")
						continue writeLoop
					}
				}
			}
			buf.WriteByte(c)
			if i++; i >= len(pat) {
				return "", fmt.Errorf("[ was not matched with a closing ]")
			}
			switch c = pat[i]; c {
			case '!', '^':
				buf.WriteByte('^')
				if i++; i >= len(pat) {
					return "", fmt.Errorf("[ was not matched with a closing ]")
				}
			}
			if c = pat[i]; c == ']' {
				buf.WriteByte(']')
				if i++; i >= len(pat) {
					return "", fmt.Errorf("[ was not matched with a closing ]")
				}
			}
			rangeStart := byte(0)
		loopBracket:
			for ; i < len(pat); i++ {
				c = pat[i]
				buf.WriteByte(c)
				switch c {
				case '\\':
					if i++; i < len(pat) {
						buf.WriteByte(pat[i])
					}
					continue
				case ']':
					break loopBracket
				}
				if rangeStart != 0 && rangeStart > c {
					return "", fmt.Errorf("invalid range: %c-%c", rangeStart, c)
				}
				if c == '-' {
					rangeStart = pat[i-1]
				} else {
					rangeStart = 0
				}
			}
			if i >= len(pat) {
				return "", fmt.Errorf("[ was not matched with a closing ]")
			}
		case '{':
			if mode&Braces == 0 {
				buf.WriteString(regexp.QuoteMeta(string(c)))
				break
			}
			innerLevel := 1
			commas := false
		peekBrace:
			for j := i + 1; j < len(pat); j++ {
				switch c := pat[j]; c {
				case '{':
					innerLevel++
				case ',':
					commas = true
				case '\\':
					j++
				case '}':
					if innerLevel--; innerLevel > 0 {
						continue
					}
					if !commas {
						break peekBrace
					}
					closingBraces = append(closingBraces, j)
					buf.WriteString("(?:")
					continue writeLoop
				}
			}
			if match := numRange.FindStringSubmatch(pat[i+1:]); len(match) == 3 {
				start, err1 := strconv.Atoi(match[1])
				end, err2 := strconv.Atoi(match[2])
				if err1 != nil || err2 != nil || start > end {
					return "", fmt.Errorf("invalid range: %q", match[0])
				}
				// TODO: can we do better here?
				buf.WriteString("(?:")
				for n := start; n <= end; n++ {
					if n > start {
						buf.WriteByte('|')
					}
					fmt.Fprintf(&buf, "%d", n)
				}
				buf.WriteByte(')')
				i += len(match[0])
				break
			}
			buf.WriteString(regexp.QuoteMeta(string(c)))
		case ',':
			if len(closingBraces) == 0 {
				buf.WriteString(regexp.QuoteMeta(string(c)))
			} else {
				buf.WriteByte('|')
			}
		case '}':
			if len(closingBraces) > 0 && closingBraces[len(closingBraces)-1] == i {
				buf.WriteByte(')')
				closingBraces = closingBraces[:len(closingBraces)-1]
			} else {
				buf.WriteString(regexp.QuoteMeta(string(c)))
			}
		default:
			if c > 128 {
				buf.WriteByte(c)
			} else {
				buf.WriteString(regexp.QuoteMeta(string(c)))
			}
		}
	}
	return buf.String(), nil
}

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

// HasMeta returns whether a string contains any unescaped pattern
// metacharacters: '*', '?', or '['. When the function returns false, the given
// pattern can only match at most one string.
//
// For example, HasMeta(`foo\*bar`) returns false, but HasMeta(`foo*bar`)
// returns true.
//
// This can be useful to avoid extra work, like TranslatePattern. Note that this
// function cannot be used to avoid QuotePattern, as backslashes are quoted by
// that function but ignored here.
func HasMeta(pat string, mode Mode) bool {
	for i := 0; i < len(pat); i++ {
		switch pat[i] {
		case '\\':
			i++
		case '*', '?', '[':
			return true
		case '{':
			if mode&Braces != 0 {
				return true
			}
		}
	}
	return false
}

// QuoteMeta returns a string that quotes all pattern metacharacters in the
// given text. The returned string is a pattern that matches the literal text.
//
// For example, QuoteMeta(`foo*bar?`) returns `foo\*bar\?`.
func QuoteMeta(pat string, mode Mode) string {
	any := false
loop:
	for _, r := range pat {
		switch r {
		case '{':
			if mode&Braces == 0 {
				continue
			}
			fallthrough
		case '*', '?', '[', '\\':
			any = true
			break loop
		}
	}
	if !any { // short-cut without a string copy
		return pat
	}
	var buf bytes.Buffer
	for _, r := range pat {
		switch r {
		case '*', '?', '[', '\\':
			buf.WriteByte('\\')
		case '{':
			if mode&Braces != 0 {
				buf.WriteByte('\\')
			}
		}
		buf.WriteRune(r)
	}
	return buf.String()
}
