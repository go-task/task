package taskfile

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

const taskignore = ".taskignore"

func GetTasksWithSources(t *ast.Taskfile) []*ast.Task {
	var tasksWithSources []*ast.Task

	for _, task := range t.Tasks.Values() {
		if len(task.Sources) > 0 {
			tasksWithSources = append(tasksWithSources, task)
		}
	}

	return tasksWithSources
}

func ReadTaskignore(l *logger.Logger, dir string, timeout time.Duration) []string {
	bytes := read(l, dir, timeout)
	globs := filterGlobs(bytes)

	return globs
}

func read(l *logger.Logger, dir string, timeout time.Duration) []byte {
	fileNode, err := NewFileNode(l, "", path.Join(dir, taskignore))
	if err != nil {
		return nil
	}

	ctx, cf := context.WithTimeout(context.Background(), timeout)
	defer cf()

	bytes, err := fileNode.Read(ctx)
	if err != nil {
		return nil
	}

	return bytes
}

func filterGlobs(bytes []byte) []string {
	if len(bytes) == 0 {
		return nil
	}

	lines := strings.Split(string(bytes), "\n")
	var validGlobs []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			validGlobs = append(validGlobs, trimmed)
		}
	}

	return validGlobs
}
