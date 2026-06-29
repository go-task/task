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

func setupExecutor(t *testing.T) *task.Executor {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(testTaskfile), 0o644))

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
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{""})

	require.ElementsMatch(t,
		[]string{"build", "deploy", "dep", "ship", "dynenum", "docs:serve"},
		values(suggs),
	)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
	require.Contains(t, descriptions(suggs), "Deploy the application")
}

func TestComplete_AliasResolvesToTaskVars(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"dep", ""})
	require.Equal(t, []string{"ENV=dev", "ENV=staging", "ENV=prod", "REGION="}, values(suggs))
	require.Equal(t, complete.DirectiveNoSpace|complete.DirectiveNoFileComp, dir)
}

func TestComplete_StaticEnum(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"deploy", ""})

	require.Equal(t, []string{"ENV=dev", "ENV=staging", "ENV=prod", "REGION="}, values(suggs))
	require.Equal(t, complete.DirectiveNoSpace|complete.DirectiveNoFileComp, dir)
}

func TestComplete_EnumRef(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, _ := complete.Complete(e, newTestFlagSet(), []string{"dynenum", ""})
	require.Equal(t, []string{"ENV=dev", "ENV=staging", "ENV=prod"}, values(suggs))
}

func TestComplete_NoRequires(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"build", ""})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_FlagValueNotConfusedWithTaskName(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--dir", "deploy", ""})
	require.ElementsMatch(t,
		[]string{"build", "deploy", "dep", "ship", "dynenum", "docs:serve"},
		values(suggs),
	)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_NamespacedTaskName(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"docs:serve", ""})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_FlagValueInlineEquals(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--output="})
	require.Equal(t, []string{"interleaved", "group", "prefixed"}, values(suggs))
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_AfterDash(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"deploy", "--", ""})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveDefault, dir)
}

func TestComplete_FlagNames(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"-"})
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
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--output", ""})
	require.Equal(t, []string{"interleaved", "group", "prefixed"}, values(suggs))
	require.Equal(t, complete.DirectiveNoFileComp, dir)
}

func TestComplete_EnumFlagValue_Sort(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, _ := complete.Complete(e, newTestFlagSet(), []string{"--sort", ""})
	require.Equal(t, []string{"default", "alphanumeric", "none"}, values(suggs))
}

func TestComplete_PathFlag_Taskfile(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--taskfile", ""})
	require.Equal(t, []string{"yml", "yaml"}, values(suggs))
	require.Equal(t, complete.DirectiveFilterFileExt, dir)
}

func TestComplete_PathFlag_Dir(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--dir", ""})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveFilterDirs, dir)
}

func TestComplete_PathFlag_Cacert(t *testing.T) {
	t.Parallel()

	e := setupExecutor(t)
	suggs, dir := complete.Complete(e, newTestFlagSet(), []string{"--cacert", ""})
	require.Empty(t, suggs)
	require.Equal(t, complete.DirectiveDefault, dir)
}

func TestComplete_NilExecutor(t *testing.T) {
	t.Parallel()

	suggs, dir := complete.Complete(nil, newTestFlagSet(), []string{"-"})
	require.NotEmpty(t, suggs)
	require.Equal(t, complete.DirectiveNoFileComp, dir)
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
