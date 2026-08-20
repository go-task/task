package complete_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/complete"
)

func newTestFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	var b bool
	var s string
	fs.BoolVarP(&b, "list-all", "a", false, "Lists all tasks")
	fs.BoolVarP(&b, "list", "l", false, "Lists tasks with descriptions")
	fs.BoolVarP(&b, "verbose", "v", false, "Verbose mode")
	fs.StringVarP(&s, "taskfile", "t", "", "Taskfile path")
	fs.StringVarP(&s, "dir", "d", "", "Run dir")
	fs.StringVarP(&s, "output", "o", "", "Output style")
	fs.StringVar(&s, "sort", "", "Sort order")
	fs.StringVar(&s, "cacert", "", "CA cert path")
	fs.StringVar(&s, "completion", "", "Generate a completion script")
	fs.StringVar(&s, "new-completion", "", "Generate a completion script")
	return fs
}

const testTaskfile = `version: '3'

vars:
  ALLOWED_ENVS:
    - dev
    - staging
    - prod

tasks:
  deploy:
    desc: Deploy the application
    aliases: [dep, ship]
    requires:
      vars:
        - name: ENV
          enum:
            - dev
            - staging
            - prod
        - REGION
    cmds:
      - 'echo {{.ENV}} {{.REGION}}'

  build:
    desc: Build it
    cmds:
      - 'echo build'

  dynenum:
    desc: Dynamic enum
    requires:
      vars:
        - name: ENV
          enum:
            ref: .ALLOWED_ENVS
    cmds:
      - 'echo {{.ENV}}'

  docs:serve:
    desc: Serve docs locally
    cmds:
      - 'echo serving'
`

const wildcardTaskfile = `version: '3'

tasks:
  wildcard-*:
    cmds:
      - 'echo {{index .MATCH 0}}'

  wildcard-*-*:
    cmds:
      - 'echo {{index .MATCH 0}}'

  '*-wildcard-*':
    cmds:
      - 'echo {{index .MATCH 0}}'

  matches-exactly-*:
    cmds:
      - 'echo {{.MATCH}}'

  start-*:
    desc: Start a service
    aliases: [s-*]
    cmds:
      - 'echo {{index .MATCH 0}}'

  build:
    desc: Build it
    cmds:
      - 'echo build'
`

func setupExecutor(t *testing.T) *task.Executor {
	t.Helper()
	return setupExecutorWith(t, testTaskfile)
}

func setupExecutorWith(t *testing.T, taskfile string) *task.Executor {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644))

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithVersionCheck(false),
	)
	require.NoError(t, e.Setup())
	return e
}

func TestComplete_TaskNames(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{""}, complete.Options{})

	require.ElementsMatch(t,
		[]string{"build", "deploy", "dep", "ship", "dynenum", "docs:serve"},
		values(suggs),
	)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
	require.Contains(t, descriptions(suggs), "Deploy the application")
}

func TestComplete_WildcardTaskNames(t *testing.T) {
	t.Parallel()

	e := setupExecutorWith(t, wildcardTaskfile)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{""}, complete.Options{})

	// Patterns are cut at their first `*`: `wildcard-*` and `wildcard-*-*`
	// collapse into one candidate, and `*-wildcard-*` leaves nothing to insert.
	require.Equal(t, []string{"build", "matches-exactly-", "start-", "s-", "wildcard-"}, values(suggs))
	require.Equal(t, complete.DirectiveNoSpace|complete.DirectiveNoFileComp, dir)
	// Without a desc, the pattern says what the prefix stands for.
	require.Contains(t, descriptions(suggs), "wildcard-*")

	suggs, _ = complete.Complete(e, newTestFlagSet(), []string{""}, complete.Options{NoDescriptions: true})
	require.Equal(t, []string{"", "", "", "", ""}, descriptions(suggs))
}

func TestComplete_AliasResolvesToTaskVars(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"dep", ""}, complete.Options{})
	require.Equal(t, []string{"ENV=dev", "ENV=staging", "ENV=prod", "REGION="}, values(suggs))
	require.Equal(t, complete.DirectiveNoSpace|complete.DirectiveNoFileComp|complete.DirectiveKeepOrder, dir)
}

func TestComplete_StaticEnum(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"deploy", ""}, complete.Options{})

	require.Equal(t, []string{"ENV=dev", "ENV=staging", "ENV=prod", "REGION="}, values(suggs))
	require.Equal(t, complete.DirectiveNoSpace|complete.DirectiveNoFileComp|complete.DirectiveKeepOrder, dir)
}

func TestComplete_EnumRef(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, _ := complete.Complete(e, newTestFlagSet(), []string{"dynenum", ""}, complete.Options{})
	require.Equal(t, []string{"ENV=dev", "ENV=staging", "ENV=prod"}, values(suggs))
}

func TestComplete_NoRequires(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"build", ""}, complete.Options{})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_FlagValueNotConfusedWithTaskName(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--dir", "deploy", ""}, complete.Options{})
	require.ElementsMatch(t,
		[]string{"build", "deploy", "dep", "ship", "dynenum", "docs:serve"},
		values(suggs),
	)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_NamespacedTaskName(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"docs:serve", ""}, complete.Options{})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_FlagValueInlineEquals(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--output="}, complete.Options{})
	// The inline form returns full `--output=value` tokens.
	require.Equal(t, []string{"--output=interleaved", "--output=group", "--output=prefixed"}, values(suggs))
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_AfterDash(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"deploy", "--", ""}, complete.Options{})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveDefault, dir)
}

func TestComplete_FlagNames(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"-"}, complete.Options{})
	require.NotEmpty(t, suggs)
	require.Equal(t, complete.DirectiveNoFileComp, dir)

	vals := values(suggs)
	require.Contains(t, vals, "--list-all")
	require.Contains(t, vals, "--taskfile")
	require.Contains(t, vals, "-a")
}

func TestComplete_EnumFlagValue_Output(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--output", ""}, complete.Options{})
	require.Equal(t, []string{"interleaved", "group", "prefixed"}, values(suggs))
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_EnumFlagValue_Sort(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, _ := complete.Complete(e, newTestFlagSet(), []string{"--sort", ""}, complete.Options{})
	require.Equal(t, []string{"default", "alphanumeric", "none"}, values(suggs))
}

func TestComplete_PathFlag_Taskfile(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--taskfile", ""}, complete.Options{})
	require.Equal(t, []string{"yml", "yaml"}, values(suggs))
	require.Equal(t, complete.DirectiveFilterFileExt, dir)
}

func TestComplete_PathFlag_Dir(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--dir", ""}, complete.Options{})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveFilterDirs, dir)
}

func TestComplete_PathFlag_Cacert(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--cacert", ""}, complete.Options{})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveDefault, dir)
}

func TestComplete_NilExecutor(t *testing.T) {
	t.Parallel()

	suggs, dir := complete.Complete(nil, newTestFlagSet(), []string{"-"}, complete.Options{})
	require.NotEmpty(t, suggs)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_NoAliases(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	opts := complete.Options{NoAliases: true}
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{""}, opts)

	require.ElementsMatch(t,
		[]string{"build", "deploy", "dynenum", "docs:serve"},
		values(suggs),
	)
	require.NotContains(t, values(suggs), "dep")
	require.NotContains(t, values(suggs), "ship")
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_NoDescriptions(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	opts := complete.Options{NoDescriptions: true}
	suggs, _ := complete.Complete(e, newTestFlagSet(), []string{""}, opts)

	require.ElementsMatch(t,
		[]string{"build", "deploy", "dep", "ship", "dynenum", "docs:serve"},
		values(suggs),
	)
	for _, d := range descriptions(suggs) {
		require.Empty(t, d)
	}
}

func TestParseOptions(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		opts, rest := complete.ParseOptions([]string{"deploy", ""})
		require.Equal(t, complete.Options{}, opts)
		require.Equal(t, []string{"deploy", ""}, rest)
	})

	t.Run("both flags", func(t *testing.T) {
		t.Parallel()
		opts, rest := complete.ParseOptions([]string{"--no-aliases", "--no-descriptions", "deploy", ""})
		require.True(t, opts.NoAliases)
		require.True(t, opts.NoDescriptions)
		require.Equal(t, []string{"deploy", ""}, rest)
	})

	t.Run("only leading flags consumed", func(t *testing.T) {
		t.Parallel()
		// A flag appearing after the user's words is left in the command line.
		opts, rest := complete.ParseOptions([]string{"deploy", "--no-aliases"})
		require.False(t, opts.NoAliases)
		require.Equal(t, []string{"deploy", "--no-aliases"}, rest)
	})
}

func TestNeedsTaskfile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args []string
		want bool
	}{
		"task name":            {[]string{""}, true},
		"partial task name":    {[]string{"bui"}, true},
		"task var":             {[]string{"deploy", ""}, true},
		"value flag then name": {[]string{"--dir", "/tmp", ""}, true},
		"flag name":            {[]string{"-"}, false},
		"long flag name":       {[]string{"--li"}, false},
		"inline flag value":    {[]string{"--output="}, false},
		"flag value":           {[]string{"--output", ""}, false},
		"path flag value":      {[]string{"--taskfile", ""}, false},
		"after dash":           {[]string{"deploy", "--", ""}, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, complete.NeedsTaskfile(tt.args, newTestFlagSet()))
		})
	}
}

// Reading the Taskfile from standard input would block until EOF, freezing the
// shell on a keystroke.
func TestNeedsTaskfile_StdinEntrypoint(t *testing.T) {
	t.Parallel()

	fs := newTestFlagSet()
	require.NoError(t, fs.Set("taskfile", "-"))
	require.False(t, complete.NeedsTaskfile([]string{""}, fs))
	require.False(t, complete.NeedsTaskfile([]string{"deploy", ""}, fs))
}

// Keeps the shells the engine offers in step with the scripts the root package
// can actually serve.
func TestCompletionShells(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"--completion", "--new-completion"} {
		suggs, dir := complete.Complete(nil, newTestFlagSet(), []string{flag, ""}, complete.Options{})
		require.Equal(t, complete.DirectiveNoFileComp, dir)
		require.NotEmpty(t, suggs)

		for _, shell := range values(suggs) {
			_, err := task.Completion(shell)
			require.NoErrorf(t, err, "%s offers %q", flag, shell)
			_, err = task.CompletionNext(shell)
			require.NoErrorf(t, err, "%s offers %q", flag, shell)
		}
	}
}

func TestWrite_Format(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	complete.Write(&buf, []complete.Suggestion{
		{Value: "deploy", Description: "Deploy the app"},
		{Value: "build"},
	}, complete.DirectiveNoSpace|complete.DirectiveNoFileComp)
	require.Equal(t, "deploy\tDeploy the app\nbuild\n:6\n", buf.String())
}

func TestWrite_EmptyWithDirective(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	complete.Write(&buf, nil, complete.DirectiveFilterDirs)
	require.Equal(t, ":16\n", buf.String())
}

func values(suggs []complete.Suggestion) []string {
	out := make([]string, 0, len(suggs))
	for _, s := range suggs {
		out = append(out, s.Value)
	}
	return out
}

func descriptions(suggs []complete.Suggestion) []string {
	out := make([]string, 0, len(suggs))
	for _, s := range suggs {
		out = append(out, s.Description)
	}
	return out
}
