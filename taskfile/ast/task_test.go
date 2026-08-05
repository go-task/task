package ast_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestTaskWildcardMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		taskName      string
		call          string
		wantMatch     bool
		wantWildcards []string
	}{
		{"build-*", "build-foo", true, []string{"foo"}},
		{"build-*-*", "build-foo-bar", true, []string{"foo", "bar"}},
		{"build-*", "test-foo", false, nil},
		// Regex metacharacters in the task name must be matched literally, not
		// interpreted as regex (and must not panic in MustCompile).
		{"c++", "c++", true, []string{}},
		{"c++", "cxx", false, nil},
		{"a.b", "axb", false, nil}, // '.' must not act as a wildcard
		{"a.b", "a.b", true, []string{}},
		{"deploy.prod", "deploy-prod", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.taskName+"/"+tt.call, func(t *testing.T) {
			t.Parallel()
			task := &ast.Task{Task: tt.taskName}
			match, wildcards := task.WildcardMatch(tt.call)
			assert.Equal(t, tt.wantMatch, match)
			assert.Equal(t, tt.wantWildcards, wildcards)
		})
	}
}
