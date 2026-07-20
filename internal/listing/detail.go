package listing

import (
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
)

func FormatRequires(req *ast.Requires) string {
	if req == nil || len(req.Vars) == 0 {
		return ""
	}
	parts := make([]string, len(req.Vars))
	for i, v := range req.Vars {
		if v.Enum != nil && len(v.Enum.Value) > 0 {
			parts[i] = v.Name + " (enum: " + strings.Join(v.Enum.Value, ", ") + ")"
		} else {
			parts[i] = v.Name
		}
	}
	return strings.Join(parts, ", ")
}

func FormatDeps(deps []*ast.Dep) string {
	if len(deps) == 0 {
		return ""
	}
	names := make([]string, len(deps))
	for i, d := range deps {
		names[i] = d.Task
	}
	return strings.Join(names, ", ")
}

func HasRequires(t *ast.Task) bool {
	return t.Requires != nil && len(t.Requires.Vars) > 0
}
