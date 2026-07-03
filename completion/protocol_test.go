// Package completion_test black-box tests the `task __complete` wire protocol —
// the candidates and directive the engine emits for a given command line. This
// replaces the old completion/tests/engine.sh with readable, table-driven Go:
// the shell wrappers only need to be smoke-tested for how they *interpret* the
// directive (see completion/tests/wrapper.*), never for the suggestion logic,
// which is fully covered here and in internal/complete.
package completion_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/complete"
)

// taskBin is the path to the task binary built once for the whole package.
var taskBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "task-completion-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	taskBin = filepath.Join(dir, "task")
	if runtime.GOOS == "windows" {
		taskBin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", taskBin, "github.com/go-task/task/v3/cmd/task").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build task binary: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

const fixtureTaskfile = `version: '3'

tasks:
  build:
    desc: Build it
  deploy:
    desc: Deploy the application
    aliases: [dep, ship]
    requires:
      vars:
        - name: ENV
          enum: [dev, staging, prod]
        - REGION
  docs:serve:
    desc: Serve docs locally
`

// completeArgs runs `task __complete <args>` in a fresh fixture directory and
// returns the offered candidate values plus the emitted directive.
func completeArgs(t *testing.T, args ...string) ([]string, complete.Directive) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(fixtureTaskfile), 0o644))

	cmd := exec.Command(taskBin, append([]string{complete.CommandName}, args...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	require.NotEmpty(t, lines, "protocol output must end with a directive line")

	last := lines[len(lines)-1]
	require.True(t, strings.HasPrefix(last, ":"), "last line must be the :<directive> line, got %q", last)
	n, err := strconv.Atoi(strings.TrimPrefix(last, ":"))
	require.NoError(t, err)

	values := make([]string, 0, len(lines)-1)
	for _, line := range lines[:len(lines)-1] {
		values = append(values, strings.SplitN(line, "\t", 2)[0])
	}
	return values, complete.Directive(n)
}

func TestProtocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		want      []string // candidate values that must be offered
		absent    []string // candidate values that must NOT be offered
		directive complete.Directive
	}{
		{
			name:      "task names and aliases",
			args:      []string{""},
			want:      []string{"build", "deploy", "dep", "ship", "docs:serve"},
			directive: complete.DirectiveNoFileComp,
		},
		{
			name:      "no-aliases drops aliases",
			args:      []string{"--no-aliases", ""},
			want:      []string{"build", "deploy"},
			absent:    []string{"dep", "ship"},
			directive: complete.DirectiveNoFileComp,
		},
		{
			name:      "flag names",
			args:      []string{"-"},
			want:      []string{"--taskfile", "--dir", "--output"},
			directive: complete.DirectiveNoFileComp,
		},
		{
			name:      "separate flag value is bare",
			args:      []string{"--output", ""},
			want:      []string{"interleaved", "group", "prefixed"},
			directive: complete.DirectiveNoFileComp,
		},
		{
			name:      "inline flag value is full form",
			args:      []string{"--output="},
			want:      []string{"--output=interleaved", "--output=group", "--output=prefixed"},
			directive: complete.DirectiveNoFileComp,
		},
		{
			name:      "sort enum values",
			args:      []string{"--sort", ""},
			want:      []string{"default", "alphanumeric", "none"},
			directive: complete.DirectiveNoFileComp,
		},
		{
			name:      "taskfile filters by extension",
			args:      []string{"--taskfile", ""},
			want:      []string{"yml", "yaml"},
			directive: complete.DirectiveFilterFileExt,
		},
		{
			name:      "dir filters to directories",
			args:      []string{"--dir", ""},
			directive: complete.DirectiveFilterDirs,
		},
		{
			name:      "task variables keep order and suppress the space",
			args:      []string{"deploy", ""},
			want:      []string{"ENV=dev", "ENV=staging", "ENV=prod", "REGION="},
			directive: complete.DirectiveNoSpace | complete.DirectiveNoFileComp | complete.DirectiveKeepOrder,
		},
		{
			name:      "after -- yields default file completion",
			args:      []string{"deploy", "--", ""},
			directive: complete.DirectiveDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			values, directive := completeArgs(t, tt.args...)
			require.Equal(t, tt.directive, directive)
			require.Subset(t, values, tt.want)
			for _, a := range tt.absent {
				require.NotContains(t, values, a)
			}
		})
	}
}
