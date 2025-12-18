package task_test

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/filepathext"
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
		executorOpts    []task.ExecutorOption
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
		task: "default",
		vars: map[string]any{},
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
		var buffer SyncBuffer

		opts := append(
			tt.executorOpts,
			task.WithStdout(&buffer),
			task.WithStderr(&buffer),
		)

		// If the test has input, create a reader for it and add it to the
		// executor options
		if tt.input != "" {
			var reader bytes.Buffer
			reader.WriteString(tt.input)
			opts = append(opts, task.WithStdin(&reader))
		}

		// Set up the task executor
		e := task.NewExecutor(opts...)

		// Create a golden fixture file for the output
		g := goldie.New(t,
			goldie.WithFixtureDir(filepath.Join(e.Dir, "testdata")),
		)

		// Call setup and check for errors
		if err := e.Setup(); tt.wantSetupError {
			require.Error(t, err)
			tt.writeFixtureErrSetup(t, g, err)
			tt.writeFixtureBuffer(t, g, buffer.buf)
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
		ctx := t.Context()
		if err := e.Run(ctx, call); tt.wantRunError {
			require.Error(t, err)
			tt.writeFixtureErrRun(t, g, err)
			tt.writeFixtureBuffer(t, g, buffer.buf)
			return
		} else {
			require.NoError(t, err)
		}

		// If the status flag is set, run the status check
		if tt.wantStatusError {
			if err := e.Status(ctx, call); err != nil {
				tt.writeFixtureStatus(t, g, err.Error())
			}
		}

		tt.writeFixtureBuffer(t, g, buffer.buf)
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
		WithExecutorOptions(
			task.WithDir("testdata/empty_task"),
		),
	)
}

func TestEmptyTaskfile(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/empty_taskfile"),
		),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestEnv(t *testing.T) {
	t.Setenv("QUX", "from_os")
	NewExecutorTest(t,
		WithName("env precedence disabled"),
		WithExecutorOptions(
			task.WithDir("testdata/env"),
			task.WithSilent(true),
		),
	)
	NewExecutorTest(t,
		WithName("env precedence enabled"),
		WithExecutorOptions(
			task.WithDir("testdata/env"),
			task.WithSilent(true),
		),
		WithExperiment(&experiments.EnvPrecedence, 1),
	)
}

func TestVars(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/vars"),
			task.WithSilent(true),
		),
	)
	NewExecutorTest(t,
		WithName("cli-var-priority-default"),
		WithExecutorOptions(
			task.WithDir("testdata/vars"),
			task.WithSilent(true),
		),
		WithTask("cli-var-priority"),
	)
	NewExecutorTest(t,
		WithName("cli-var-priority-override"),
		WithExecutorOptions(
			task.WithDir("testdata/vars"),
			task.WithSilent(true),
		),
		WithTask("cli-var-priority"),
		WithVar("CLI_VAR", "from_cli"),
	)
}

func TestRequires(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithName("required var missing"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("missing-var"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("required var ok"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("missing-var"),
		WithVar("FOO", "bar"),
	)
	NewExecutorTest(t,
		WithName("fails validation"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var"),
		WithVar("ENV", "dev"),
		WithVar("FOO", "bar"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("passes validation"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var"),
		WithVar("FOO", "one"),
		WithVar("ENV", "dev"),
	)
	NewExecutorTest(t,
		WithName("required var missing + fails validation"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("required var missing + fails validation"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var-dynamic"),
		WithVar("FOO", "one"),
		WithVar("ENV", "dev"),
	)
	NewExecutorTest(t,
		WithName("require before compile"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("require-before-compile"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("var defined in task"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("var-defined-in-task"),
	)
}

// TODO: mock fs
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

	for _, dir := range []string{dir, subdir} {
		for _, test := range tests {
			NewExecutorTest(t,
				WithName(fmt.Sprintf("%s-%s", dir, test)),
				WithExecutorOptions(
					task.WithDir(dir),
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
		WithExecutorOptions(
			task.WithDir("testdata/concurrency"),
			task.WithConcurrency(1),
		),
		WithPostProcessFn(PPSortedLines),
	)
}

func TestParams(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/params"),
			task.WithSilent(true),
		),
		WithPostProcessFn(PPSortedLines),
	)
}

func TestDeps(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/deps"),
			task.WithSilent(true),
		),
		WithPostProcessFn(PPSortedLines),
	)
}

// TODO: mock fs
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
		WithExecutorOptions(
			task.WithDir(dir),
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
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithSilent(true),
		),
		WithTask("gen-bar"),
	)
	// gen-silent-baz is marked as being silent, and should only produce output
	// if e.Verbose is set to true.
	NewExecutorTest(t,
		WithName("run gen-baz silent"),
		WithExecutorOptions(
			task.WithDir(dir),
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
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithSilent(true),
		),
		WithTask("gen-bar"),
	)
	// Run gen-bar a third time, to make sure we've triggered the status check.
	NewExecutorTest(t,
		WithName("run gen-bar 3 silent"),
		WithExecutorOptions(
			task.WithDir(dir),
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
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithSilent(true),
		),
		WithTask("gen-bar"),
	)
	// all: not up-to-date
	NewExecutorTest(t,
		WithName("run gen-foo 2"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("gen-foo"),
	)
	// status: not up-to-date
	NewExecutorTest(t,
		WithName("run gen-foo 3"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("gen-foo"),
	)
	// sources: not up-to-date
	NewExecutorTest(t,
		WithName("run gen-bar 5"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("gen-bar"),
	)
	// all: up-to-date
	NewExecutorTest(t,
		WithName("run gen-bar 6"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("gen-bar"),
	)
	// sources: not up-to-date, no output produced.
	NewExecutorTest(t,
		WithName("run gen-baz 2"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("gen-silent-baz"),
	)
	// up-to-date, no output produced
	NewExecutorTest(t,
		WithName("run gen-baz 3"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("gen-silent-baz"),
	)
	// up-to-date, output produced due to Verbose mode.
	NewExecutorTest(t,
		WithName("run gen-baz 4 verbose"),
		WithExecutorOptions(
			task.WithDir(dir),
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
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("foo"),
	)
	NewExecutorTest(t,
		WithName("a precondition was not met"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("impossible"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("precondition in dependency fails the task"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("depends_on_impossible"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("precondition in cmd fails the task"),
		WithExecutorOptions(
			task.WithDir(dir),
		),
		WithTask("executes_failing_task_as_cmd"),
		WithRunError(),
	)
}

func TestAlias(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("alias"),
		WithExecutorOptions(
			task.WithDir("testdata/alias"),
		),
		WithTask("f"),
	)

	NewExecutorTest(t,
		WithName("duplicate alias"),
		WithExecutorOptions(
			task.WithDir("testdata/alias"),
		),
		WithTask("x"),
		WithRunError(),
	)

	NewExecutorTest(t,
		WithName("alias summary"),
		WithExecutorOptions(
			task.WithDir("testdata/alias"),
			task.WithSummary(true),
		),
		WithTask("f"),
	)
}

func TestSummaryWithVarsAndRequires(t *testing.T) {
	t.Parallel()

	// Test basic case from prompt.md - vars and requires
	NewExecutorTest(t,
		WithName("vars-and-requires"),
		WithExecutorOptions(
			task.WithDir("testdata/summary-vars-requires"),
			task.WithSummary(true),
		),
		WithTask("mytask"),
	)

	// Test with shell variables
	NewExecutorTest(t,
		WithName("shell-vars"),
		WithExecutorOptions(
			task.WithDir("testdata/summary-vars-requires"),
			task.WithSummary(true),
		),
		WithTask("with-sh-var"),
	)
}

func TestLabel(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("up to date"),
		WithExecutorOptions(
			task.WithDir("testdata/label_uptodate"),
		),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("summary"),
		WithExecutorOptions(
			task.WithDir("testdata/label_summary"),
			task.WithSummary(true),
		),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("status"),
		WithExecutorOptions(
			task.WithDir("testdata/label_status"),
		),
		WithTask("foo"),
		WithStatusError(),
	)

	NewExecutorTest(t,
		WithName("var"),
		WithExecutorOptions(
			task.WithDir("testdata/label_var"),
		),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("label in summary"),
		WithExecutorOptions(
			task.WithDir("testdata/label_summary"),
		),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("label in error"),
		WithExecutorOptions(
			task.WithDir("testdata/label_error"),
		),
		WithTask("foo"),
		WithRunError(),
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
				WithExecutorOptions(
					task.WithDir("testdata/prompt"),
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
		WithExecutorOptions(
			task.WithDir("testdata/prompt"),
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
		WithExecutorOptions(
			task.WithDir("testdata/prompt"),
			task.WithAssumeTerm(true),
			task.WithAssumeYes(true),
		),
		WithTask("foo"),
		WithInput("\n"),
	)

	NewExecutorTest(t,
		WithName("task should raise errors.TaskCancelledError"),
		WithExecutorOptions(
			task.WithDir("testdata/prompt"),
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
			WithExecutorOptions(
				task.WithDir("testdata/for/cmds"),
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
			WithExecutorOptions(
				task.WithDir("testdata/for/deps"),
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
			WithExecutorOptions(
				task.WithDir("testdata/var_references"),
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
			WithExecutorOptions(
				task.WithDir(fmt.Sprintf("testdata/var_inheritance/v3/%s", test.name)),
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
		WithExecutorOptions(
			task.WithDir("testdata/fuzzy"),
		),
		WithTask("instal"),
		WithRunError(),
	)

	NewExecutorTest(t,
		WithName("not-fuzzy"),
		WithExecutorOptions(
			task.WithDir("testdata/fuzzy"),
		),
		WithTask("install"),
	)

	NewExecutorTest(t,
		WithName("intern"),
		WithExecutorOptions(
			task.WithDir("testdata/fuzzy"),
		),
		WithTask("intern"),
		WithRunError(),
	)
}

func TestIncludeChecksum(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("correct"),
		WithExecutorOptions(
			task.WithDir("testdata/includes_checksum/correct"),
		),
	)

	NewExecutorTest(t,
		WithName("incorrect"),
		WithExecutorOptions(
			task.WithDir("testdata/includes_checksum/incorrect"),
		),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestFailfast(t *testing.T) {
	t.Parallel()

	t.Run("Default", func(t *testing.T) {
		t.Parallel()

		NewExecutorTest(t,
			WithName("default"),
			WithExecutorOptions(
				task.WithDir("testdata/failfast/default"),
				task.WithSilent(true),
			),
			WithPostProcessFn(PPSortedLines),
			WithRunError(),
		)
	})

	t.Run("Option", func(t *testing.T) {
		t.Parallel()

		NewExecutorTest(t,
			WithName("default"),
			WithExecutorOptions(
				task.WithDir("testdata/failfast/default"),
				task.WithSilent(true),
				task.WithFailfast(true),
			),
			WithPostProcessFn(PPSortedLines),
			WithRunError(),
		)
	})

	t.Run("Task", func(t *testing.T) {
		t.Parallel()

		NewExecutorTest(t,
			WithName("task"),
			WithExecutorOptions(
				task.WithDir("testdata/failfast/task"),
				task.WithSilent(true),
			),
			WithPostProcessFn(PPSortedLines),
			WithRunError(),
		)
	})
}
