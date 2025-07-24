package task_test

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile"
	"github.com/go-task/task/v3/taskfile/ast"
)

type (
	// A ExecutorTestOption is a function that configures an [ExecutorTest].
	ExecutorTestOption interface {
		applyToExecutorTest(*ExecutorTest)
	}
	// A ExecutorTest is a test wrapper around a [task.Executor] to make it easy
	// to write tests for tasks. See [NewExecutorTest] for information on
	// creating and running ExecutorTests. These tests use fixture files to
	// assert whether the result of a task is correct. If Task's behavior has
	// been changed, the fixture files can be updated by running `task
	// gen:fixtures`.
	ExecutorTest struct {
		TaskTest
		task            string
		vars            map[string]any
		input           string
		nodeDir         string
		nodeEntrypoint  string
		nodeInsecure    bool
		readerOpts      []taskfile.ReaderOption
		executorOpts    []task.ExecutorOption
		wantReaderError bool
		wantSetupError  bool
		wantRunError    bool
		wantStatusError bool
	}
)

// NewExecutorTest sets up a new [task.Executor] with the given options and runs
// a task with the given [ExecutorTestOption]s. The output of the task is
// written to a set of fixture files depending on the configuration of the test.
func NewExecutorTest(t *testing.T, opts ...ExecutorTestOption) {
	t.Helper()
	tt := &ExecutorTest{
		task:    "default",
		vars:    map[string]any{},
		nodeDir: ".",
		TaskTest: TaskTest{
			experiments:         map[*experiments.Experiment]int{},
			fixtureTemplateData: map[string]any{},
		},
	}
	// Apply the functional options
	for _, opt := range opts {
		opt.applyToExecutorTest(tt)
	}
	// Enable any experiments that have been set
	for x, v := range tt.experiments {
		prev := *x
		*x = experiments.Experiment{
			Name:          prev.Name,
			AllowedValues: []int{v},
			Value:         v,
		}
		t.Cleanup(func() {
			*x = prev
		})
	}
	tt.run(t)
}

// Functional options

// WithInput tells the test to create a reader with the given input. This can be
// used to simulate user input when a task requires it.
func WithInput(input string) ExecutorTestOption {
	return &inputTestOption{input}
}

type inputTestOption struct {
	input string
}

func (opt *inputTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.input = opt.input
}

// WithRunError tells the test to expect an error during the run phase of the
// task execution. A fixture will be created with the output of any errors.
func WithRunError() ExecutorTestOption {
	return &runErrorTestOption{}
}

type runErrorTestOption struct{}

func (opt *runErrorTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.wantRunError = true
}

// WithStatusError tells the test to make an additional call to
// [task.Executor.Status] after the task has been run. A fixture will be created
// with the output of any errors.
func WithStatusError() ExecutorTestOption {
	return &statusErrorTestOption{}
}

type statusErrorTestOption struct{}

func (opt *statusErrorTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.wantStatusError = true
}

// Helpers

// writeFixtureErrRun is a wrapper for writing the output of an error during the
// run phase of the task to a fixture file.
func (tt *ExecutorTest) writeFixtureErrRun(
	t *testing.T,
	g *goldie.Goldie,
	err error,
) {
	t.Helper()
	tt.writeFixture(t, g, "err-run", []byte(err.Error()))
}

// writeFixtureStatus is a wrapper for writing the output of an error when
// making an additional call to [task.Executor.Status] to a fixture file.
func (tt *ExecutorTest) writeFixtureStatus(
	t *testing.T,
	g *goldie.Goldie,
	status string,
) {
	t.Helper()
	tt.writeFixture(t, g, "err-status", []byte(status))
}

// run is the main function for running the test. It sets up the task executor,
// runs the task, and writes the output to a fixture file.
func (tt *ExecutorTest) run(t *testing.T) {
	t.Helper()
	f := func(t *testing.T) {
		t.Helper()
		var buf bytes.Buffer
		ctx := context.Background()

		// Create a new root node for the given entrypoint
		node, err := taskfile.NewRootNode(
			tt.nodeEntrypoint,
			tt.nodeDir,
			tt.nodeInsecure,
		)
		require.NoError(t, err)

		// Create a golden fixture file for the output
		g := goldie.New(t,
			goldie.WithFixtureDir(filepath.Join(node.Dir(), "testdata")),
		)

		// Set up a temporary directory for the taskfile reader and task executor
		tempDir, err := task.NewTempDir(node.Dir())
		require.NoError(t, err)
		tt.readerOpts = append(tt.readerOpts, taskfile.WithTempDir(tempDir.Remote))

		// Set up the taskfile reader
		reader := taskfile.NewReader(tt.readerOpts...)
		graph, err := reader.Read(ctx, node)
		if tt.wantReaderError {
			require.Error(t, err)
			tt.writeFixtureErrReader(t, g, err)
			tt.writeFixtureBuffer(t, g, buf)
			return
		} else {
			require.NoError(t, err)
		}

		executorOpts := slices.Concat(
			// Apply the node directory and temp directory to the executor options
			// by default, but allow them to by overridden by the test options
			[]task.ExecutorOption{
				task.WithDir(node.Dir()),
				task.WithTempDir(tempDir),
			},
			// Apply the executor options from the test
			tt.executorOpts,
			// Force the input/output streams to be set to the test buffer
			[]task.ExecutorOption{
				task.WithStdout(&buf),
				task.WithStderr(&buf),
			},
		)

		// If the test has input, create a reader for it and add it to the
		// executor options
		if tt.input != "" {
			var reader bytes.Buffer
			reader.WriteString(tt.input)
			executorOpts = append(executorOpts, task.WithStdin(&reader))
		}

		// Set up the task executor
		executor, err := task.NewExecutor(graph, executorOpts...)
		if tt.wantSetupError {
			require.Error(t, err)
			tt.writeFixtureErrSetup(t, g, err)
			tt.writeFixtureBuffer(t, g, buf)
			return
		} else {
			require.NoError(t, err)
		}

		// Create the task call
		vars := ast.NewVars()
		for key, value := range tt.vars {
			vars.Set(key, ast.Var{Value: value})
		}
		call := &task.Call{
			Task: tt.task,
			Vars: vars,
		}

		// Run the task and check for errors
		if err := executor.Run(ctx, call); tt.wantRunError {
			require.Error(t, err)
			tt.writeFixtureErrRun(t, g, err)
			tt.writeFixtureBuffer(t, g, buf)
			return
		} else {
			require.NoError(t, err)
		}

		// If the status flag is set, run the status check
		if tt.wantStatusError {
			if err := executor.Status(ctx, call); err != nil {
				tt.writeFixtureStatus(t, g, err.Error())
			}
		}

		tt.writeFixtureBuffer(t, g, buf)
	}

	// Run the test (with a name if it has one)
	if tt.name != "" {
		t.Run(tt.name, f)
	} else {
		f(t)
	}
}

func TestEmptyTask(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithNodeDir("testdata/empty_task"),
		WithExecutorOptions(),
	)
}

func TestEmptyTaskfile(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithNodeDir("testdata/empty_taskfile"),
		WithReaderError(),
		WithFixtureTemplating(),
	)
}

func TestEnv(t *testing.T) {
	t.Setenv("QUX", "from_os")
	NewExecutorTest(t,
		WithName("env precedence disabled"),
		WithNodeDir("testdata/env"),
		WithExecutorOptions(
			task.WithSilent(true),
		),
	)
	NewExecutorTest(t,
		WithName("env precedence enabled"),
		WithNodeDir("testdata/env"),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithExperiment(&experiments.EnvPrecedence, 1),
	)
}

func TestVars(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithNodeDir("testdata/vars"),
		WithExecutorOptions(
			task.WithSilent(true),
		),
	)
}

func TestRequires(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithName("required var missing"),
		WithNodeDir("testdata/requires"),
		WithTask("missing-var"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("required var ok"),
		WithNodeDir("testdata/requires"),
		WithTask("missing-var"),
		WithVar("FOO", "bar"),
	)
	NewExecutorTest(t,
		WithName("fails validation"),
		WithNodeDir("testdata/requires"),
		WithTask("validation-var"),
		WithVar("ENV", "dev"),
		WithVar("FOO", "bar"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("passes validation"),
		WithNodeDir("testdata/requires"),
		WithTask("validation-var"),
		WithVar("FOO", "one"),
		WithVar("ENV", "dev"),
	)
	NewExecutorTest(t,
		WithName("required var missing + fails validation"),
		WithNodeDir("testdata/requires"),
		WithTask("validation-var"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("required var missing + fails validation"),
		WithNodeDir("testdata/requires"),
		WithTask("validation-var-dynamic"),
		WithVar("FOO", "one"),
		WithVar("ENV", "dev"),
	)
	NewExecutorTest(t,
		WithName("require before compile"),
		WithNodeDir("testdata/requires"),
		WithTask("require-before-compile"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("var defined in task"),
		WithNodeDir("testdata/requires"),
		WithTask("var-defined-in-task"),
	)
}

func TestSpecialVars(t *testing.T) {
	t.Parallel()

	const dir = "testdata/special_vars"
	const subdir = "testdata/special_vars/subdir"

	tests := []string{
		// Root
		"print-task",
		"print-root-dir",
		"print-taskfile",
		"print-taskfile-dir",
		"print-task-dir",
		// Included
		"included:print-task",
		"included:print-root-dir",
		"included:print-taskfile",
		"included:print-taskfile-dir",
	}

	for _, executorDir := range []string{dir, subdir} {
		for _, test := range tests {
			name := fmt.Sprintf("%s-%s", executorDir, test)
			NewExecutorTest(t,
				WithName(name),
				WithNodeDir(executorDir),
				WithExecutorOptions(
					task.WithSilent(true),
					task.WithVersionCheck(true),
				),
				WithTask(test),
				WithFixtureTemplating(),
			)
		}
	}
}

func TestConcurrency(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithNodeDir("testdata/concurrency"),
		WithExecutorOptions(
			task.WithConcurrency(1),
		),
		WithPostProcessFn(PPSortedLines),
	)
}

func TestParams(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithNodeDir("testdata/params"),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithPostProcessFn(PPSortedLines),
	)
}

func TestDeps(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithNodeDir("testdata/deps"),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithPostProcessFn(PPSortedLines),
	)
}

func TestStatus(t *testing.T) {
	t.Parallel()

	const dir = "testdata/status"

	files := []string{
		"foo.txt",
		"bar.txt",
		"baz.txt",
	}

	for _, f := range files {
		path := filepathext.SmartJoin(dir, f)
		_ = os.Remove(path)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("File should not exist: %v", err)
		}
	}

	// gen-foo creates foo.txt, and will always fail it's status check.
	NewExecutorTest(t,
		WithName("run gen-foo 1 silent"),
		WithNodeDir(dir),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithTask("gen-foo"),
	)
	// gen-foo creates bar.txt, and will pass its status-check the 3. time it
	// is run. It creates bar.txt, but also lists it as its source. So, the checksum
	// for the file won't match before after the second run as we the file
	// only exists after the first run.
	NewExecutorTest(t,
		WithName("run gen-bar 1 silent"),
		WithNodeDir(dir),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithTask("gen-bar"),
	)
	// gen-silent-baz is marked as being silent, and should only produce output
	// if e.Verbose is set to true.
	NewExecutorTest(t,
		WithName("run gen-baz silent"),
		WithNodeDir(dir),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithTask("gen-silent-baz"),
	)

	for _, f := range files {
		if _, err := os.Stat(filepathext.SmartJoin(dir, f)); err != nil {
			t.Errorf("File should exist: %v", err)
		}
	}

	// Run gen-bar a second time to produce a checksum file that matches bar.txt
	NewExecutorTest(t,
		WithName("run gen-bar 2 silent"),
		WithNodeDir(dir),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithTask("gen-bar"),
	)
	// Run gen-bar a third time, to make sure we've triggered the status check.
	NewExecutorTest(t,
		WithName("run gen-bar 3 silent"),
		WithNodeDir(dir),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithTask("gen-bar"),
	)

	// Now, let's remove source file, and run the task again to to prepare
	// for the next test.
	err := os.Remove(filepathext.SmartJoin(dir, "bar.txt"))
	require.NoError(t, err)
	NewExecutorTest(t,
		WithName("run gen-bar 4 silent"),
		WithNodeDir(dir),
		WithExecutorOptions(
			task.WithSilent(true),
		),
		WithTask("gen-bar"),
	)
	// all: not up-to-date
	NewExecutorTest(t,
		WithName("run gen-foo 2"),
		WithNodeDir(dir),
		WithTask("gen-foo"),
	)
	// status: not up-to-date
	NewExecutorTest(t,
		WithName("run gen-foo 3"),
		WithNodeDir(dir),
		WithTask("gen-foo"),
	)
	// sources: not up-to-date
	NewExecutorTest(t,
		WithName("run gen-bar 5"),
		WithNodeDir(dir),
		WithTask("gen-bar"),
	)
	// all: up-to-date
	NewExecutorTest(t,
		WithName("run gen-bar 6"),
		WithNodeDir(dir),
		WithTask("gen-bar"),
	)
	// sources: not up-to-date, no output produced.
	NewExecutorTest(t,
		WithName("run gen-baz 2"),
		WithNodeDir(dir),
		WithTask("gen-silent-baz"),
	)
	// up-to-date, no output produced
	NewExecutorTest(t,
		WithName("run gen-baz 3"),
		WithNodeDir(dir),
		WithTask("gen-silent-baz"),
	)
	// up-to-date, output produced due to Verbose mode.
	NewExecutorTest(t,
		WithName("run gen-baz 4 verbose"),
		WithNodeDir(dir),
		WithExecutorOptions(
			task.WithVerbose(true),
		),
		WithTask("gen-silent-baz"),
		WithFixtureTemplating(),
	)
}

func TestPrecondition(t *testing.T) {
	t.Parallel()
	const dir = "testdata/precondition"
	NewExecutorTest(t,
		WithName("a precondition has been met"),
		WithNodeDir(dir),
		WithTask("foo"),
	)
	NewExecutorTest(t,
		WithName("a precondition was not met"),
		WithNodeDir(dir),
		WithTask("impossible"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("precondition in dependency fails the task"),
		WithNodeDir(dir),
		WithTask("depends_on_impossible"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("precondition in cmd fails the task"),
		WithNodeDir(dir),
		WithTask("executes_failing_task_as_cmd"),
		WithRunError(),
	)
}

func TestAlias(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("alias"),
		WithNodeDir("testdata/alias"),
		WithTask("f"),
	)

	NewExecutorTest(t,
		WithName("duplicate alias"),
		WithNodeDir("testdata/alias"),
		WithTask("x"),
		WithRunError(),
	)

	NewExecutorTest(t,
		WithName("alias summary"),
		WithNodeDir("testdata/alias"),
		WithExecutorOptions(
			task.WithSummary(true),
		),
		WithTask("f"),
	)
}

func TestLabel(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("up to date"),
		WithNodeDir("testdata/label_uptodate"),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("summary"),
		WithNodeDir("testdata/label_summary"),
		WithExecutorOptions(
			task.WithSummary(true),
		),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("status"),
		WithNodeDir("testdata/label_status"),
		WithTask("foo"),
		WithStatusError(),
	)

	NewExecutorTest(t,
		WithName("var"),
		WithNodeDir("testdata/label_var"),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("label in summary"),
		WithNodeDir("testdata/label_summary"),
		WithTask("foo"),
	)
}

func TestPromptInSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"test short approval", "y\n", false},
		{"test long approval", "yes\n", false},
		{"test uppercase approval", "Y\n", false},
		{"test stops task", "n\n", true},
		{"test junk value stops task", "foobar\n", true},
		{"test Enter stops task", "\n", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			opts := []ExecutorTestOption{
				WithName(test.name),
				WithNodeDir("testdata/prompt"),
				WithExecutorOptions(
					task.WithAssumeTerm(true),
				),
				WithTask("foo"),
				WithInput(test.input),
			}
			if test.wantError {
				opts = append(opts, WithRunError())
			}
			NewExecutorTest(t, opts...)
		})
	}
}

func TestPromptWithIndirectTask(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithNodeDir("testdata/prompt"),
		WithExecutorOptions(
			task.WithAssumeTerm(true),
		),
		WithTask("bar"),
		WithInput("y\n"),
	)
}

func TestPromptAssumeYes(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("--yes flag should skip prompt"),
		WithNodeDir("testdata/prompt"),
		WithExecutorOptions(
			task.WithAssumeTerm(true),
			task.WithAssumeYes(true),
		),
		WithTask("foo"),
		WithInput("\n"),
	)

	NewExecutorTest(t,
		WithName("task should raise errors.TaskCancelledError"),
		WithNodeDir("testdata/prompt"),
		WithExecutorOptions(
			task.WithAssumeTerm(true),
		),
		WithTask("foo"),
		WithInput("\n"),
		WithRunError(),
	)
}

func TestForCmds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "loop-explicit"},
		{name: "loop-matrix"},
		{name: "loop-matrix-ref"},
		{
			name:    "loop-matrix-ref-error",
			wantErr: true,
		},
		{name: "loop-sources"},
		{name: "loop-sources-glob"},
		{name: "loop-generates"},
		{name: "loop-generates-glob"},
		{name: "loop-vars"},
		{name: "loop-vars-sh"},
		{name: "loop-task"},
		{name: "loop-task-as"},
		{name: "loop-different-tasks"},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithNodeDir("testdata/for/cmds"),
			WithExecutorOptions(
				task.WithSilent(true),
				task.WithForce(true),
			),
			WithTask(test.name),
			WithFixtureTemplating(),
		}
		if test.wantErr {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestForDeps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "loop-explicit"},
		{name: "loop-matrix"},
		{name: "loop-matrix-ref"},
		{
			name:    "loop-matrix-ref-error",
			wantErr: true,
		},
		{name: "loop-sources"},
		{name: "loop-sources-glob"},
		{name: "loop-generates"},
		{name: "loop-generates-glob"},
		{name: "loop-vars"},
		{name: "loop-vars-sh"},
		{name: "loop-task"},
		{name: "loop-task-as"},
		{name: "loop-different-tasks"},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithNodeDir("testdata/for/deps"),
			WithExecutorOptions(
				task.WithSilent(true),
				task.WithForce(true),
				// Force output of each dep to be grouped together to prevent interleaving
				task.WithOutputStyle(ast.Output{Name: "group"}),
			),
			WithTask(test.name),
			WithFixtureTemplating(),
			WithPostProcessFn(PPSortedLines),
		}
		if test.wantErr {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call string
	}{
		{
			name: "reference in command",
			call: "ref-cmd",
		},
		{
			name: "reference in dependency",
			call: "ref-dep",
		},
		{
			name: "reference using templating resolver",
			call: "ref-resolver",
		},
		{
			name: "reference using templating resolver and dynamic var",
			call: "ref-resolver-sh",
		},
	}

	for _, test := range tests {
		NewExecutorTest(t,
			WithName(test.name),
			WithNodeDir("testdata/var_references"),
			WithExecutorOptions(
				task.WithSilent(true),
				task.WithForce(true),
			),
			WithTask(cmp.Or(test.call, "default")),
		)
	}
}

func TestVarInheritance(t *testing.T) {
	enableExperimentForTest(t, &experiments.EnvPrecedence, 1)
	tests := []struct {
		name string
		call string
	}{
		{name: "shell"},
		{name: "entrypoint-global-dotenv"},
		{name: "entrypoint-global-vars"},
		// We can't send env vars to a called task, so the env var is not overridden
		{name: "entrypoint-task-call-vars"},
		// Dotenv doesn't set variables
		{name: "entrypoint-task-call-dotenv"},
		{name: "entrypoint-task-call-task-vars"},
		// Dotenv doesn't set variables
		{name: "entrypoint-task-dotenv"},
		{name: "entrypoint-task-vars"},
		// {
		// 	// Dotenv not currently allowed in included taskfiles
		// 	name: "included-global-dotenv",
		// 	want: "included-global-dotenv\nincluded-global-dotenv\n",
		// },
		{
			name: "included-global-vars",
			call: "included",
		},
		{
			// We can't send env vars to a called task, so the env var is not overridden
			name: "included-task-call-vars",
			call: "included",
		},
		{
			// Dotenv doesn't set variables
			// Dotenv not currently allowed in included taskfiles (but doesn't error in a task)
			name: "included-task-call-dotenv",
			call: "included",
		},
		{
			name: "included-task-call-task-vars",
			call: "included",
		},
		{
			// Dotenv doesn't set variables
			// Somehow dotenv is working here!
			name: "included-task-dotenv",
			call: "included",
		},
		{
			name: "included-task-vars",
			call: "included",
		},
	}

	t.Setenv("VAR", "shell")
	t.Setenv("ENV", "shell")
	for _, test := range tests {
		NewExecutorTest(t,
			WithName(test.name),
			WithNodeDir(fmt.Sprintf("testdata/var_inheritance/v3/%s", test.name)),
			WithExecutorOptions(
				task.WithSilent(true),
				task.WithForce(true),
			),
			WithTask(cmp.Or(test.call, "default")),
		)
	}
}

func TestFuzzyModel(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("fuzzy"),
		WithNodeDir("testdata/fuzzy"),
		WithTask("instal"),
		WithRunError(),
	)

	NewExecutorTest(t,
		WithName("not-fuzzy"),
		WithNodeDir("testdata/fuzzy"),
		WithTask("install"),
	)

	NewExecutorTest(t,
		WithName("intern"),
		WithNodeDir("testdata/fuzzy"),
		WithTask("intern"),
		WithRunError(),
	)
}

func TestIncludeChecksum(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("correct"),
		WithNodeDir("testdata/includes_checksum/correct"),
	)

	NewExecutorTest(t,
		WithName("incorrect"),
		WithNodeDir("testdata/includes_checksum/incorrect"),
		WithReaderError(),
		WithFixtureTemplating(),
	)
}

func TestWildcard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		call    string
		wantErr bool
	}{
		{
			name: "basic wildcard",
			call: "wildcard-foo",
		},
		{
			name: "double wildcard",
			call: "foo-wildcard-bar",
		},
		{
			name: "store wildcard",
			call: "start-foo",
		},
		{
			name: "matches exactly",
			call: "matches-exactly-*",
		},
		{
			name:    "no matches",
			call:    "no-match",
			wantErr: true,
		},
		{
			name: "multiple matches",
			call: "wildcard-foo-bar",
		},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithNodeDir("testdata/wildcards"),
			WithExecutorOptions(
				task.WithSilent(true),
				task.WithForce(true),
			),
			WithTask(test.call),
		}
		if test.wantErr {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}
