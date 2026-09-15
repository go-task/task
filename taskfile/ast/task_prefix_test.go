package ast_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		target   string
		expected bool
	}{
		{
			name:     "Exact match single segment",
			input:    "build",
			target:   "build",
			expected: true,
		},
		{
			name:     "Prefix match single segment",
			input:    "b",
			target:   "build",
			expected: true,
		},
		{
			name:     "Non-matching single segment",
			input:    "c",
			target:   "build",
			expected: false,
		},
		{
			name:     "Exact multi-segment match",
			input:    "api:openapi:export",
			target:   "api:openapi:export",
			expected: true,
		},
		{
			name:     "Segment abbreviation m=n",
			input:    "a:o:e",
			target:   "api:openapi:export",
			expected: true,
		},
		{
			name:     "Segment abbreviation partial m=n",
			input:    "api:open:ex",
			target:   "api:openapi:export",
			expected: true,
		},
		{
			name:     "Shorter input segments m < n",
			input:    "api:o",
			target:   "api:openapi:export",
			expected: true,
		},
		{
			name:     "Shorter input segments single segment m < n",
			input:    "a",
			target:   "api:openapi:export",
			expected: true,
		},
		{
			name:     "Mismatch in first segment",
			input:    "d:o:e",
			target:   "api:openapi:export",
			expected: false,
		},
		{
			name:     "Mismatch in middle segment",
			input:    "a:x:e",
			target:   "api:openapi:export",
			expected: false,
		},
		{
			name:     "Mismatch in last segment",
			input:    "a:o:x",
			target:   "api:openapi:export",
			expected: false,
		},
		{
			name:     "Longer input segments m > n",
			input:    "a:o:e:extra",
			target:   "api:openapi:export",
			expected: false,
		},
		{
			name:     "Empty input",
			input:    "",
			target:   "build",
			expected: false,
		},
		{
			name:     "Empty target",
			input:    "build",
			target:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ast.MatchesPrefix(tt.input, tt.target)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestTaskMatchesPrefixWithAliases(t *testing.T) {
	task := &ast.Task{
		Task:    "docker:build:production",
		Aliases: []string{"d:b:prod", "prod-build"},
	}

	assert.True(t, task.MatchesPrefix("d:b:p"))
	assert.True(t, task.MatchesPrefix("docker:b"))
	assert.True(t, task.MatchesPrefix("prod-b"))
	assert.False(t, task.MatchesPrefix("docker:push"))
	assert.False(t, task.MatchesPrefix("staging"))
}
