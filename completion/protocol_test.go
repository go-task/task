// Black-box tests of the `task __complete` wire protocol. How each shell
// wrapper interprets the directive is smoke-tested in completion/tests/.
package completion_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/complete"
)

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
	if out, err := exec.CommandContext(context.Background(), "go", "build", "-o", taskBin, "github.com/go-task/task/v3/cmd/task").CombinedOutput(); err != nil {
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

// completeArgs runs `task __complete <args>` in a fresh fixture directory.
func completeArgs(t *testing.T, args ...string) ([]string, complete.Directive) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(fixtureTaskfile), 0o644))

	cmd := exec.CommandContext(t.Context(), taskBin, append([]string{complete.CommandName}, args...)...) //nolint:gosec
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err)

	return parseProtocol(t, out)
}

func parseProtocol(t *testing.T, out []byte) ([]string, complete.Directive) {
	t.Helper()

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

// --sort is the flag deciding how the Taskfile is read with a visible order.
func TestProtocol_SortFlagIsApplied(t *testing.T) {
	t.Parallel()

	const taskfile = `version: '3'

tasks:
  zebra:
    desc: Declared first, last alphabetically
  alpha:
    desc: Declared last, first alphabetically
`
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644))

	sorted, _ := completeInDir(t, dir, nil, "")
	require.Equal(t, []string{"alpha", "zebra"}, sorted)

	declared, _ := completeInDir(t, dir, nil, "--sort", "none", "")
	require.Equal(t, []string{"zebra", "alpha"}, declared)
}

func TestProtocol_ExperimentGatedFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(fixtureTaskfile), 0o644))

	values, directive := completeInDir(t, dir, []string{"TASK_X_GENTLE_FORCE=1"}, "--force-all", "")
	require.Equal(t, complete.DirectiveNoFileComp, directive)
	require.Subset(t, values, []string{"build", "deploy"})
}

// Downloading an uncached remote include would freeze the shell for up to
// --timeout and prompt for trust.
func TestProtocol_RemoteIncludeStaysOffline(t *testing.T) {
	t.Parallel()

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		<-r.Context().Done()
	}))
	defer srv.Close()

	taskfile := fmt.Sprintf(`version: '3'

includes:
  remote: %s/Taskfile.yml

tasks:
  build:
    desc: Build it
`, srv.URL)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644))

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	// A fresh cache dir leaves a download as the only way to resolve the
	// include; the insecure opt-in keeps the plain-HTTP server from being
	// rejected before it.
	cmd := exec.CommandContext(ctx, taskBin, complete.CommandName, "") //nolint:gosec
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"TASK_REMOTE_DIR="+t.TempDir(),
		"TASK_REMOTE_INSECURE=1",
	)
	out, err := cmd.Output()
	require.NoError(t, err, "completion must not hang on a remote include")

	_, directive := parseProtocol(t, out)
	require.Equal(t, complete.DirectiveNoFileComp, directive)
	require.Zero(t, hits.Load(), "completion must not reach the network")
}

// `--taskfile -` would otherwise read the Taskfile from the terminal.
func TestProtocol_StdinEntrypointDoesNotHang(t *testing.T) {
	t.Parallel()

	// An unwritten pipe: reading it would block until the context expires.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		r.Close()
		w.Close()
	})

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, taskBin, complete.CommandName, "-t", "-", "") //nolint:gosec
	cmd.Dir = t.TempDir()
	cmd.Stdin = r
	out, err := cmd.Output()
	require.NoError(t, err, "completion must not read the Taskfile from stdin")

	_, directive := parseProtocol(t, out)
	require.Equal(t, complete.DirectiveNoFileComp, directive)
}

func TestProtocol_WildcardTaskNames(t *testing.T) {
	t.Parallel()

	values, directive := completeInDir(t, filepath.Join("..", "testdata", "wildcards"), nil, "")
	require.Equal(t, complete.DirectiveNoSpace|complete.DirectiveNoFileComp, directive)
	require.Subset(t, values, []string{"start-", "s-", "wildcard-", "matches-exactly-"})
	for _, v := range values {
		require.NotEmpty(t, v)
		require.NotContains(t, v, "*")
	}
}

// completeInDir runs `task __complete <args>` in dir, with env appended to the
// current environment.
func completeInDir(t *testing.T, dir string, env []string, args ...string) ([]string, complete.Directive) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), taskBin, append([]string{complete.CommandName}, args...)...) //nolint:gosec
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	require.NoError(t, err)

	return parseProtocol(t, out)
}

// Keeps the shells the engine offers in step with the scripts the root package
// can actually serve.
func TestCompletionShells(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"--completion", "--new-completion"} {
		shells, directive := completeArgs(t, flag, "")
		require.Equal(t, complete.DirectiveNoFileComp, directive)
		require.NotEmpty(t, shells)

		for _, shell := range shells {
			_, err := task.Completion(shell)
			require.NoErrorf(t, err, "%s offers %q", flag, shell)
			_, err = task.CompletionNext(shell)
			require.NoErrorf(t, err, "%s offers %q", flag, shell)
		}
	}
}
