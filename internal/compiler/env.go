package compiler

import (
	"os"
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
)

// GetEnviron the all return all environment variables encapsulated on a
// ast.Vars
func GetEnviron() *ast.Vars {
	m := &ast.Vars{}
	for _, e := range os.Environ() {
		keyVal := strings.SplitN(e, "=", 2)
		key, val := keyVal[0], keyVal[1]
		m.Set(key, ast.Var{Value: val})
	}
	return m
}
