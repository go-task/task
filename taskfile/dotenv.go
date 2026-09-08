package taskfile

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/elliotchance/orderedmap/v3"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

func Dotenv(vars *ast.Vars, tf *ast.Taskfile, dir string) (*ast.Vars, error) {
	env := ast.NewVars()
	cache := &templater.Cache{Vars: vars}

	for _, dotEnvPath := range tf.Dotenv {
		dotEnvPath = templater.Replace(dotEnvPath, cache)
		if dotEnvPath == "" {
			continue
		}
		dotEnvPath = filepathext.SmartJoin(dir, dotEnvPath)

		if _, err := os.Stat(dotEnvPath); os.IsNotExist(err) {
			continue
		}

		envs, err := ReadDotenvOrdered(dotEnvPath)
		if err != nil {
			return nil, fmt.Errorf("error reading env file %s: %w", dotEnvPath, err)
		}
		for key, value := range envs.AllFromFront() {
			if _, ok := env.Get(key); !ok {
				env.Set(key, ast.Var{Value: value})
			}
		}
	}

	return env, nil
}

// ReadDotenvOrdered reads and parses a dotenv file, returning its variables in
// the order they are declared in the file. This is important because Task
// re-resolves each dotenv variable's value with its own {{.VAR}} templater,
// referencing the values of variables declared earlier in the same file. If
// the variables were provided out of order, a variable that depends on
// another declared above it in the file might resolve before its dependency
// does, depending on iteration order.
//
// The parsing logic below (through dotenvUnescapeCharsRegex) is adapted from
// github.com/joho/godotenv v1.5.1's parser.go, replacing its use of a plain
// map with an ordered map so that declaration order is preserved.
//
// godotenv is Copyright (c) 2013 John Barton and is distributed under the
// MIT License: https://github.com/joho/godotenv/blob/v1.5.1/LICENCE
func ReadDotenvOrdered(path string) (*orderedmap.OrderedMap[string, string], error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := orderedmap.NewOrderedMap[string, string]()
	if err := parseDotenvBytes(src, out); err != nil {
		return nil, err
	}
	return out, nil
}

const (
	dotenvCharComment       = '#'
	dotenvPrefixSingleQuote = '\''
	dotenvPrefixDoubleQuote = '"'
	dotenvExportPrefix      = "export"
)

func parseDotenvBytes(src []byte, out *orderedmap.OrderedMap[string, string]) error {
	src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	cutset := src
	for {
		cutset = dotenvStatementStart(cutset)
		if cutset == nil {
			break
		}

		key, left, err := dotenvKeyName(cutset)
		if err != nil {
			return err
		}

		value, left, err := dotenvVarValue(left, out)
		if err != nil {
			return err
		}

		out.Set(key, value)
		cutset = left
	}

	return nil
}

// dotenvStatementStart returns the position of the next statement, skipping
// any comment lines or leading whitespace.
func dotenvStatementStart(src []byte) []byte {
	pos := bytes.IndexFunc(src, func(r rune) bool { return !unicode.IsSpace(r) })
	if pos == -1 {
		return nil
	}

	src = src[pos:]
	if src[0] != dotenvCharComment {
		return src
	}

	pos = bytes.IndexRune(src, '\n')
	if pos == -1 {
		return nil
	}

	return dotenvStatementStart(src[pos:])
}

// dotenvKeyName locates and parses a key name and returns the rest of the slice.
func dotenvKeyName(src []byte) (key string, cutset []byte, err error) {
	src = bytes.TrimLeftFunc(src, dotenvIsSpace)
	if trimmed, ok := bytes.CutPrefix(src, []byte(dotenvExportPrefix)); ok {
		if bytes.IndexFunc(trimmed, dotenvIsSpace) == 0 {
			src = bytes.TrimLeftFunc(trimmed, dotenvIsSpace)
		}
	}

	offset := 0
loop:
	for i, char := range src {
		rchar := rune(char)
		if dotenvIsSpace(rchar) {
			continue
		}

		switch char {
		case '=', ':':
			key = string(src[0:i])
			offset = i + 1
			break loop
		case '_':
		default:
			if unicode.IsLetter(rchar) || unicode.IsNumber(rchar) || rchar == '.' {
				continue
			}
			return "", nil, fmt.Errorf(
				`unexpected character %q in variable name near %q`,
				string(char), string(src))
		}
	}

	if len(src) == 0 {
		return "", nil, fmt.Errorf("zero length string")
	}

	key = strings.TrimRightFunc(key, unicode.IsSpace)
	cutset = bytes.TrimLeftFunc(src[offset:], dotenvIsSpace)
	return key, cutset, nil
}

// dotenvVarValue extracts a variable value and returns the rest of the slice.
func dotenvVarValue(src []byte, vars *orderedmap.OrderedMap[string, string]) (value string, rest []byte, err error) {
	quote, hasPrefix := dotenvQuotePrefix(src)
	if !hasPrefix {
		endOfLine := bytes.IndexFunc(src, dotenvIsLineEnd)

		if endOfLine == -1 {
			endOfLine = len(src)
			if endOfLine == 0 {
				return "", nil, nil
			}
		}

		line := []rune(string(src[0:endOfLine]))

		endOfVar := len(line)
		if endOfVar == 0 {
			return "", src[endOfLine:], nil
		}

		for i := endOfVar - 1; i >= 0; i-- {
			if line[i] == dotenvCharComment && i > 0 {
				if dotenvIsSpace(line[i-1]) {
					endOfVar = i
					break
				}
			}
		}

		trimmed := strings.TrimFunc(string(line[0:endOfVar]), dotenvIsSpace)

		return dotenvExpandVariables(trimmed, vars), src[endOfLine:], nil
	}

	for i := 1; i < len(src); i++ {
		if char := src[i]; char != quote {
			continue
		}

		if prevChar := src[i-1]; prevChar == '\\' {
			continue
		}

		trimFunc := func(r rune) bool { return r == rune(quote) }
		value = string(bytes.TrimLeftFunc(bytes.TrimRightFunc(src[0:i], trimFunc), trimFunc))
		if quote == dotenvPrefixDoubleQuote {
			value = dotenvExpandVariables(dotenvExpandEscapes(value), vars)
		}

		return value, src[i+1:], nil
	}

	valEndIndex := bytes.IndexRune(src, '\n')
	if valEndIndex == -1 {
		valEndIndex = len(src)
	}

	return "", nil, fmt.Errorf("unterminated quoted value %s", src[:valEndIndex])
}

func dotenvExpandEscapes(str string) string {
	out := dotenvEscapeRegex.ReplaceAllStringFunc(str, func(match string) string {
		c := match[1:]
		switch c {
		case "n":
			return "\n"
		case "r":
			return "\r"
		default:
			return match
		}
	})
	return dotenvUnescapeCharsRegex.ReplaceAllString(out, "$1")
}

func dotenvExpandVariables(v string, m *orderedmap.OrderedMap[string, string]) string {
	return dotenvExpandVarRegex.ReplaceAllStringFunc(v, func(s string) string {
		submatch := dotenvExpandVarRegex.FindStringSubmatch(s)
		if submatch == nil {
			return s
		}
		if submatch[1] == "\\" || submatch[2] == "(" {
			return submatch[0][1:]
		} else if submatch[4] != "" {
			return m.GetOrDefault(submatch[4], "")
		}
		return s
	})
}

func dotenvQuotePrefix(src []byte) (prefix byte, isQuoted bool) {
	if len(src) == 0 {
		return 0, false
	}
	switch prefix := src[0]; prefix {
	case dotenvPrefixDoubleQuote, dotenvPrefixSingleQuote:
		return prefix, true
	default:
		return 0, false
	}
}

// dotenvIsSpace reports whether the rune is a space character but not a line
// break character. This differs from unicode.IsSpace, which also treats line
// breaks as space.
func dotenvIsSpace(r rune) bool {
	switch r {
	case '\t', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	return false
}

func dotenvIsLineEnd(r rune) bool {
	return r == '\n' || r == '\r'
}

var (
	dotenvEscapeRegex        = regexp.MustCompile(`\\.`)
	dotenvExpandVarRegex     = regexp.MustCompile(`(\\)?(\$)(\()?\{?([A-Z0-9_]+)?\}?`)
	dotenvUnescapeCharsRegex = regexp.MustCompile(`\\([^$])`)
)
