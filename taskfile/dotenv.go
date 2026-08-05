package taskfile

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/joho/godotenv"

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

		envs, err := ReadDotenv(dotEnvPath)
		if err != nil {
			return nil, fmt.Errorf("error reading env file %s: %w", dotEnvPath, err)
		}
		for key, value := range envs.All() {
			if _, ok := env.Get(key); !ok {
				env.Set(key, value)
			}
		}
	}

	return env, nil
}

// ReadDotenv reads a dotenv file while preserving the variable order from the file.
func ReadDotenv(filename string) (*ast.Vars, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	envMap, err := godotenv.UnmarshalBytes(src)
	if err != nil {
		return nil, err
	}

	env := ast.NewVars()
	seen := make(map[string]bool, len(envMap))
	for _, key := range dotenvKeyOrder(src) {
		value, ok := envMap[key]
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		env.Set(key, ast.Var{Value: value})
	}

	if len(seen) < len(envMap) {
		missing := make([]string, 0, len(envMap)-len(seen))
		for key := range envMap {
			if !seen[key] {
				missing = append(missing, key)
			}
		}
		sort.Strings(missing)
		for _, key := range missing {
			env.Set(key, ast.Var{Value: envMap[key]})
		}
	}

	return env, nil
}

func dotenvKeyOrder(src []byte) []string {
	src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	keys := []string{}

	for {
		src = dotenvStatementStart(src)
		if src == nil {
			return keys
		}

		key, rest := dotenvLocateKey(src)
		if key != "" {
			keys = append(keys, key)
		}
		src = dotenvSkipValue(rest)
	}
}

func dotenvStatementStart(src []byte) []byte {
	for {
		pos := bytes.IndexFunc(src, func(r rune) bool {
			return !unicode.IsSpace(r)
		})
		if pos == -1 {
			return nil
		}

		src = src[pos:]
		if src[0] != '#' {
			return src
		}

		pos = bytes.IndexFunc(src, dotenvIsLineEnd)
		if pos == -1 {
			return nil
		}
		src = src[pos:]
	}
}

func dotenvLocateKey(src []byte) (string, []byte) {
	src = bytes.TrimLeftFunc(src, dotenvIsSpace)
	if bytes.HasPrefix(src, []byte("export")) {
		trimmed := bytes.TrimPrefix(src, []byte("export"))
		if bytes.IndexFunc(trimmed, dotenvIsSpace) == 0 {
			src = bytes.TrimLeftFunc(trimmed, dotenvIsSpace)
		}
	}

	for i, char := range src {
		rchar := rune(char)
		if dotenvIsSpace(rchar) {
			continue
		}

		switch char {
		case '=', ':':
			key := strings.TrimRightFunc(string(src[:i]), unicode.IsSpace)
			return key, bytes.TrimLeftFunc(src[i+1:], dotenvIsSpace)
		case '_':
		default:
			if unicode.IsLetter(rchar) || unicode.IsNumber(rchar) || rchar == '.' {
				continue
			}
			return "", dotenvSkipValue(src)
		}
	}

	return "", nil
}

func dotenvSkipValue(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}

	switch quote := src[0]; quote {
	case '\'', '"':
		for i := 1; i < len(src); i++ {
			if src[i] == quote && src[i-1] != '\\' {
				return src[i+1:]
			}
		}
		return nil
	default:
		pos := bytes.IndexFunc(src, dotenvIsLineEnd)
		if pos == -1 {
			return nil
		}
		return src[pos:]
	}
}

func dotenvIsSpace(r rune) bool {
	switch r {
	case '\t', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	default:
		return false
	}
}

func dotenvIsLineEnd(r rune) bool {
	return r == '\n' || r == '\r'
}
