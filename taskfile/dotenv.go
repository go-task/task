package taskfile

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// ReadEnvFile reads a dotenv file and returns an ordered Vars map.
// Unlike godotenv.Read, this preserves the declaration order of variables as
// they appear in the file, ensuring that nested {{.VAR}} template references
// expand deterministically.
func ReadEnvFile(path string) (*ast.Vars, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading env file %s: %w", path, err)
	}

	// godotenv.Parse handles $VAR expansion and all quoting rules. It reads
	// line by line internally, so $VAR references resolve in file order. The
	// returned map loses that order, which is why we re-scan for key names.
	envMap, err := godotenv.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("error reading env file %s: %w", path, err)
	}

	vars := ast.NewVars()
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimLeft(scanner.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Handle "export KEY=VALUE" syntax
		line = strings.TrimPrefix(line, "export")
		line = strings.TrimLeft(line, " \t")
		// Extract key name (everything before the first '=')
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimRight(line[:idx], " \t")
		if key == "" {
			continue
		}
		value, ok := envMap[key]
		if !ok {
			continue
		}
		if _, exists := vars.Get(key); !exists {
			vars.Set(key, ast.Var{Value: value})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading env file %s: %w", path, err)
	}
	return vars, nil
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

		fileVars, err := ReadEnvFile(dotEnvPath)
		if err != nil {
			return nil, err
		}
		for k, v := range fileVars.All() {
			if _, ok := env.Get(k); !ok {
				env.Set(k, v)
			}
		}
	}

	return env, nil
}
