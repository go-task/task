package taskfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// DotenvKeyValue represents a key-value pair from a dotenv file.
type DotenvKeyValue struct {
	Key   string
	Value string
}

// ReadDotenvOrdered reads a dotenv file and returns key-value pairs in the
// order they appear in the file. This is important because Go maps have
// non-deterministic iteration order, which can cause race conditions when
// variables reference each other with template syntax like {{.VAR}}.
func ReadDotenvOrdered(path string) ([]DotenvKeyValue, error) {
	// Use godotenv to parse the file (handles quotes, escaping, etc.)
	envMap, err := godotenv.Read(path)
	if err != nil {
		return nil, err
	}

	// Read the file again to get keys in order
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var orderedKeys []string
	seenKeys := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle "export VAR=value" syntax
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
			line = strings.TrimSpace(line)
		}

		// Extract the key (before = or :)
		var key string
		if idx := strings.IndexAny(line, "=:"); idx > 0 {
			key = strings.TrimSpace(line[:idx])
		}

		// Only add if it's a valid key we found in godotenv's output
		// and we haven't seen it before (first occurrence wins)
		if key != "" && !seenKeys[key] {
			if _, exists := envMap[key]; exists {
				orderedKeys = append(orderedKeys, key)
				seenKeys[key] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Build ordered result using godotenv's parsed values
	result := make([]DotenvKeyValue, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		result = append(result, DotenvKeyValue{
			Key:   key,
			Value: envMap[key],
		})
	}

	return result, nil
}

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
		for _, kv := range envs {
			if _, ok := env.Get(kv.Key); !ok {
				env.Set(kv.Key, ast.Var{Value: kv.Value})
			}
		}
	}

	return env, nil
}
