package task_test

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io/fs"
	rand "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/errors"
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
		tasks           []string
		vars            map[string]any
		input           string
		executorOpts    []task.ExecutorOption
		wantSetupError  bool
		wantRunError    bool
		wantStatusError bool
		noRun           bool
		assertFns       []func(t *testing.T, r *ExecutorTestResult)
	}
	// An ExecutorTestResult carries the outcome of an [ExecutorTest] run. It is
	// passed to any assertion functions registered with [WithAssert], for
	// checks that a golden fixture can't express, such as an error's concrete
	// type, timing bounds, or the [task.Executor]'s internal state.
	ExecutorTestResult struct {
		Executor *task.Executor
		Output   string
		Err      error
		Duration time.Duration
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

// WithTasks sets the names of multiple tasks to run in a single call to
// [task.Executor.Run]. Use this instead of [WithTask] when the test needs to
// call more than one task at once (e.g. to test summaries spanning several
// tasks).
func WithTasks(tasks ...string) ExecutorTestOption {
	return &tasksTestOption{tasks: tasks}
}

type tasksTestOption struct {
	tasks []string
}

func (opt *tasksTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.tasks = opt.tasks
}

// WithNoRun tells the test to stop after a successful setup, without calling
// [task.Executor.Run]. This is useful for tests that only care about the
// state of the [task.Executor] (or its parsed Taskfile) after setup, and for
// tests that construct an [task.Executor] but never actually call a task. No
// output fixture is written, since no task is run.
func WithNoRun() ExecutorTestOption {
	return &noRunTestOption{}
}

type noRunTestOption struct{}

func (opt *noRunTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.noRun = true
}

// WithAssert registers a function to run custom assertions against the
// [ExecutorTestResult] of the test, in addition to the usual error and golden
// fixture checks. This is useful for assertions that a golden fixture can't
// express, such as an error's concrete type, timing bounds, or the
// [task.Executor]'s internal state. This can be called multiple times to add
// more than one assertion function.
func WithAssert(fn func(t *testing.T, r *ExecutorTestResult)) ExecutorTestOption {
	return &assertTestOption{fn: fn}
}

type assertTestOption struct {
	fn func(t *testing.T, r *ExecutorTestResult)
}

func (opt *assertTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.assertFns = append(t.assertFns, opt.fn)
}

// Helpers

// SyncBuffer is a threadsafe buffer for testing.
// Some times replace stdout/stderr with a buffer to capture output.
// stdout and stderr are threadsafe, but a regular bytes.Buffer is not.
// Using this instead helps prevents race conditions with output.
type SyncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (sb *SyncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

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
			goldie.WithEqualFn(NormalizedEqual),
		)

		// runAsserts runs any functions registered with WithAssert against the
		// current outcome of the test.
		runAsserts := func(result *ExecutorTestResult) {
			t.Helper()
			for _, fn := range tt.assertFns {
				fn(t, result)
			}
		}

		// Call setup and check for errors
		if err := e.Setup(); tt.wantSetupError {
			require.Error(t, err)
			runAsserts(&ExecutorTestResult{Executor: e, Err: err})
			tt.writeFixtureErrSetup(t, g, err)
			tt.writeFixtureBuffer(t, g, buffer.buf)
			return
		} else {
			require.NoError(t, err)
		}

		// If the test doesn't want to run a task, stop here. There's no
		// output, so no fixture is written.
		if tt.noRun {
			runAsserts(&ExecutorTestResult{Executor: e})
			return
		}

		// Create the task call(s)
		vars := ast.NewVars()
		for key, value := range tt.vars {
			vars.Set(key, ast.Var{Value: value})
		}
		taskNames := tt.tasks
		if len(taskNames) == 0 {
			taskNames = []string{tt.task}
		}
		calls := make([]*task.Call, 0, len(taskNames))
		for _, name := range taskNames {
			calls = append(calls, &task.Call{Task: name, Vars: vars})
		}

		// Run the task and check for errors
		ctx := t.Context()
		start := time.Now()
		err := e.Run(ctx, calls...)
		duration := time.Since(start)
		if tt.wantRunError {
			require.Error(t, err)
			runAsserts(&ExecutorTestResult{Executor: e, Output: buffer.buf.String(), Err: err, Duration: duration})
			tt.writeFixtureErrRun(t, g, err)
			tt.writeFixtureBuffer(t, g, buffer.buf)
			return
		} else {
			require.NoError(t, err)
		}
		runAsserts(&ExecutorTestResult{Executor: e, Output: buffer.buf.String(), Duration: duration})

		// If the status flag is set, run the status check
		if tt.wantStatusError {
			if err := e.Status(ctx, calls[0]); err != nil {
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

func TestSecretVars(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithName("secret vars are masked in logs"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
		),
		WithTask("test-secret-masking"),
	)
	NewExecutorTest(t,
		WithName("multiple secrets masked"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
		),
		WithTask("test-multiple-secrets"),
	)
	NewExecutorTest(t,
		WithName("mixed secret and public vars"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
		),
		WithTask("test-mixed"),
	)
	NewExecutorTest(t,
		WithName("deferred command with secrets"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
		),
		WithTask("test-deferred-secret"),
	)
	NewExecutorTest(t,
		WithName("env secret limitation"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
		),
		WithTask("test-env-secret-limitation"),
	)
	NewExecutorTest(t,
		WithName("secret vars are masked in summary"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
			task.WithSummary(true),
		),
		WithTask("test-secret-masking"),
	)
	NewExecutorTest(t,
		WithName("dynamic secret masked in verbose"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
			task.WithVerbose(true),
		),
		WithTask("test-dynamic-secret-verbose"),
	)
	NewExecutorTest(t,
		WithName("secret key order independent"),
		WithExecutorOptions(
			task.WithDir("testdata/secrets"),
		),
		WithTask("test-secret-key-order"),
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
	NewExecutorTest(t,
		WithName("enum ref - passes validation"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var-ref"),
		WithVar("ENV", "dev"),
	)
	NewExecutorTest(t,
		WithName("enum ref - fails validation"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var-ref"),
		WithVar("ENV", "invalid"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("enum ref - ref to non-list"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var-ref-invalid"),
		WithVar("VALUE", "test"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("enum ref - ref to nonexistent var"),
		WithExecutorOptions(
			task.WithDir("testdata/requires"),
		),
		WithTask("validation-var-ref-nonexistent"),
		WithVar("ENV", "dev"),
		WithRunError(),
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
		"print-root-taskfile",
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
			// Deps run in parallel and, with the default interleaved output,
			// their sub-line writes (echo writes the content and the newline
			// separately) can interleave into a garbled buffer. Group output
			// flushes each command atomically, keeping every line intact. The
			// set of lines asserted (via PPSortedLines) is unchanged.
			task.WithOutputStyle(ast.Output{Name: "group"}),
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

func TestPrefix(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("up to date"),
		WithExecutorOptions(
			task.WithDir("testdata/prefix_uptodate"),
			task.WithOutputStyle(ast.Output{Name: "prefixed"}),
		),
		WithTask("foo"),
	)

	NewExecutorTest(t,
		WithName("up to dat with no output style"),
		WithExecutorOptions(
			task.WithDir("testdata/prefix_uptodate"),
		),
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
		{name: "loop-matrix-ref-computed"},
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
			WithExperiment(&experiments.EnvPrecedence, 1),
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

// writeFile writes content to a file, creating any intermediate directories.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepathext.SmartJoin(dir, name), []byte(content), 0o644))
}

// gitignoreStep writes a set of files then runs the task once, capturing its
// output as a golden fixture named run.
type gitignoreStep struct {
	write map[string]string
	run   string
}

// gitignoreSeq drives a checksum task through a sequence of runs against a
// fixture dir. create seeds runtime files (removed on cleanup); restore resets
// tracked files to their committed content on cleanup; artifacts are
// task-produced files to delete on cleanup.
type gitignoreSeq struct {
	dir       string
	task      string
	create    map[string]string
	restore   map[string]string
	artifacts []string
	steps     []gitignoreStep
}

func (s gitignoreSeq) run(t *testing.T) {
	t.Helper()
	cleanup := func() {
		// The fixture manages its own .git marker so that gitignore filtering
		// resolves a repo root regardless of the build source: an in-tree
		// checkout would otherwise inherit the go-task .git, but a GitHub
		// source tarball has none, which would silently disable filtering and
		// break the golden fixtures.
		_ = os.RemoveAll(filepathext.SmartJoin(s.dir, ".git"))
		_ = os.RemoveAll(filepathext.SmartJoin(s.dir, ".task"))
		for name := range s.create {
			_ = os.Remove(filepathext.SmartJoin(s.dir, name))
		}
		for _, name := range s.artifacts {
			_ = os.Remove(filepathext.SmartJoin(s.dir, name))
		}
		for name, content := range s.restore {
			writeFile(t, s.dir, name, content)
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	require.NoError(t, os.MkdirAll(filepathext.SmartJoin(s.dir, ".git"), 0o755))
	for name, content := range s.create {
		writeFile(t, s.dir, name, content)
	}
	for _, step := range s.steps {
		for name, content := range step.write {
			writeFile(t, s.dir, name, content)
		}
		NewExecutorTest(t,
			WithName(step.run),
			WithExecutorOptions(task.WithDir(s.dir)),
			WithTask(s.task),
		)
	}
}

func TestGitignoreChecksum(t *testing.T) { //nolint:paralleltest // shares testdata/gitignore and mutates fixture files
	gitignoreSeq{
		dir:       "testdata/gitignore",
		task:      "build",
		create:    map[string]string{"ignored.txt": "ignored\n"},
		restore:   map[string]string{"source.txt": "source content\n"},
		artifacts: []string{"generated.txt"},
		steps: []gitignoreStep{
			{run: "first run"},
			{run: "up to date"},
			{run: "ignored file modified", write: map[string]string{"ignored.txt": "ignored modified\n"}},
			{run: "source file modified", write: map[string]string{"source.txt": "source modified\n"}},
		},
	}.run(t)
}

// TestGitignoreNegation checks that a `!pattern` in a nested .gitignore
// re-includes a file excluded by a parent .gitignore.
func TestGitignoreNegation(t *testing.T) { //nolint:paralleltest // mutates fixture files
	gitignoreSeq{
		dir:    "testdata/gitignore_negation",
		task:   "build",
		create: map[string]string{"sub/debug.log": "debug\n", "sub/other.log": "other\n"},
		steps: []gitignoreStep{
			{run: "first run"},
			{run: "up to date"},
			{run: "ignored file modified", write: map[string]string{"sub/other.log": "other modified\n"}},
			{run: "reincluded file modified", write: map[string]string{"sub/debug.log": "debug modified\n"}},
		},
	}.run(t)
}

// TestGitignoreNested checks that a .gitignore in a subdirectory below the task
// dir is honored when its files are reached by a deep glob.
func TestGitignoreNested(t *testing.T) { //nolint:paralleltest // mutates fixture files
	gitignoreSeq{
		dir:     "testdata/gitignore_nested",
		task:    "build",
		create:  map[string]string{"sub/secret.dat": "secret\n"},
		restore: map[string]string{"sub/keep.txt": "keep\n"},
		steps: []gitignoreStep{
			{run: "first run"},
			{run: "up to date"},
			{run: "ignored file modified", write: map[string]string{"sub/secret.dat": "secret modified\n"}},
			{run: "source file modified", write: map[string]string{"sub/keep.txt": "keep modified\n"}},
		},
	}.run(t)
}

// TestGitignoreIncluded checks that a top-level use_gitignore in an included
// Taskfile is propagated onto its tasks during merge.
func TestGitignoreIncluded(t *testing.T) { //nolint:paralleltest // mutates fixture files
	gitignoreSeq{
		dir:    "testdata/gitignore_included",
		task:   "included:build",
		create: map[string]string{"ignored.txt": "ignored\n"},
		steps: []gitignoreStep{
			{run: "first run"},
			{run: "up to date"},
			{run: "ignored file modified", write: map[string]string{"ignored.txt": "ignored modified\n"}},
		},
	}.run(t)
}

// TestGitignoreIncludedOverride checks that an explicit use_gitignore: false in
// an included Taskfile is preserved even when the root Taskfile sets it to true.
func TestGitignoreIncludedOverride(t *testing.T) { //nolint:paralleltest // mutates fixture files
	gitignoreSeq{
		dir:    "testdata/gitignore_included_override",
		task:   "included:build",
		create: map[string]string{"ignored.txt": "ignored\n"},
		steps: []gitignoreStep{
			{run: "first run"},
			{run: "up to date"},
			{run: "ignored file modified", write: map[string]string{"ignored.txt": "ignored modified\n"}},
		},
	}.run(t)
}

func TestIncludeSilent(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("include-taskfile-silent"),
		WithExecutorOptions(
			task.WithDir("testdata/includes_silent"),
		),
		WithTask("default"),
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

func TestIf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		task    string
		vars    map[string]any
		verbose bool
	}{
		// Basic command-level if
		{name: "cmd-if-true", task: "cmd-if-true"},
		{name: "cmd-if-false", task: "cmd-if-false"},

		// Task-level if
		{name: "task-if-true", task: "task-if-true"},
		{name: "task-if-false", task: "task-if-false", verbose: true},

		// Task call with if
		{name: "task-call-if-true", task: "task-call-if-true"},
		{name: "task-call-if-false", task: "task-call-if-false", verbose: true},

		// Go template conditions
		{name: "template-eq-true", task: "template-eq-true"},
		{name: "template-eq-false", task: "template-eq-false", verbose: true},
		{name: "template-ne", task: "template-ne"},
		{name: "template-bool-true", task: "template-bool-true"},
		{name: "template-bool-false", task: "template-bool-false"},
		{name: "template-direct-true", task: "template-direct-true"},
		{name: "template-direct-false", task: "template-direct-false"},
		{name: "template-and", task: "template-and"},
		{name: "template-or", task: "template-or"},

		// CLI variable override
		{name: "template-cli-var", task: "template-cli-var", vars: map[string]any{"MY_VAR": "yes"}},

		// Task-level if with template
		{name: "task-level-template", task: "task-level-template"},
		{name: "task-level-template-false", task: "task-level-template-false", verbose: true},

		// For loop with if
		{name: "if-in-for-loop", task: "if-in-for-loop", verbose: true},

		// Task-level if with dynamic variable
		{name: "task-if-dynamic-true", task: "task-if-dynamic-true"},
		{name: "task-if-dynamic-false", task: "task-if-dynamic-false", verbose: true},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir("testdata/if"),
				task.WithSilent(true),
				task.WithVerbose(test.verbose),
			),
			WithTask(test.task),
		}
		if test.vars != nil {
			for k, v := range test.vars {
				opts = append(opts, WithVar(k, v))
			}
		}
		NewExecutorTest(t, opts...)
	}
}

func TestIncludes(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes")),
	)
}

func TestIncludesMultiLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_multi_level")),
	)
}

func TestIncludesEmptyMain(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_empty")),
		WithTask("included:default"),
	)
}

func TestIncludesDependencies(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_deps")),
	)
}

func TestIncludesCallingRoot(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_call_root_task")),
		WithTask("included:call-root"),
	)
}

func TestIncludesOptional(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_optional")),
	)
}

func TestIncludesFromCustomTaskfile(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/includes_yaml"),
			task.WithEntrypoint("testdata/includes_yaml/Custom.ext"),
		),
	)
}

func TestIncludesShadowedDefault(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_shadowed_default")),
		WithTask("included"),
	)
}

func TestIncludesUnshadowedDefault(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_unshadowed_default")),
		WithTask("included"),
	)
}

func TestIncludesRemote(t *testing.T) {
	dir := "testdata/includes_remote"
	os.RemoveAll(filepath.Join(dir, ".task", "remote"))

	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	tcs := []struct {
		firstRemote  string
		secondRemote string
	}{
		{
			firstRemote:  srv.URL + "/first/Taskfile.yml",
			secondRemote: srv.URL + "/first/second/Taskfile.yml",
		},
		{
			firstRemote:  srv.URL + "/first/Taskfile.yml",
			secondRemote: "./second/Taskfile.yml",
		},
		{
			firstRemote:  srv.URL + "/first/",
			secondRemote: srv.URL + "/first/second/",
		},
	}

	taskCalls := []*task.Call{
		{Task: "first:write-file"},
		{Task: "first:second:write-file"},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Setenv("FIRST_REMOTE_URL", tc.firstRemote)
			t.Setenv("SECOND_REMOTE_URL", tc.secondRemote)

			var buff SyncBuffer

			// Extract host from server URL for trust testing
			parsedURL, err := url.Parse(srv.URL)
			require.NoError(t, err)
			trustedHost := parsedURL.Host

			executors := []struct {
				name     string
				executor *task.Executor
			}{
				{
					name: "online, always download",
					executor: task.NewExecutor(
						task.WithDir(dir),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithTimeout(time.Minute),
						task.WithInsecure(true),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithVerbose(true),

						// Without caching
						task.WithAssumeYes(true),
						task.WithDownload(true),
					),
				},
				{
					name: "offline, use cache",
					executor: task.NewExecutor(
						task.WithDir(dir),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithTimeout(time.Minute),
						task.WithInsecure(true),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithVerbose(true),

						// With caching
						task.WithAssumeYes(false),
						task.WithDownload(false),
						task.WithOffline(true),
					),
				},
				{
					name: "with trusted hosts, no prompts",
					executor: task.NewExecutor(
						task.WithDir(dir),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithTimeout(time.Minute),
						task.WithInsecure(true),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithVerbose(true),

						// With trusted hosts
						task.WithTrustedHosts([]string{trustedHost}),
						task.WithDownload(true),
					),
				},
			}

			for _, e := range executors {
				t.Run(e.name, func(t *testing.T) {
					require.NoError(t, e.executor.Setup())

					for k, taskCall := range taskCalls {
						t.Run(taskCall.Task, func(t *testing.T) {
							expectedContent := fmt.Sprint(rand.Int64()) //nolint:gosec
							t.Setenv("CONTENT", expectedContent)

							outputFile := fmt.Sprintf("%d.%d.txt", i, k)
							t.Setenv("OUTPUT_FILE", outputFile)

							path := filepath.Join(dir, outputFile)
							require.NoError(t, os.RemoveAll(path))

							require.NoError(t, e.executor.Run(t.Context(), taskCall))

							actualContent, err := os.ReadFile(path)
							require.NoError(t, err)
							assert.Equal(t, expectedContent, strings.TrimSpace(string(actualContent)))
						})
					}
				})
			}

			t.Log("\noutput:\n", buff.buf.String())
		})
	}
}

func TestIncludesHttp(t *testing.T) { //nolint:paralleltest // sets INCLUDE_ROOT per iteration
	dir, err := filepath.Abs("testdata/includes_http")
	require.NoError(t, err)

	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	t.Cleanup(func() {
		// This test fills the .task/remote directory with cache entries because the include URL
		// is different on every test due to the dynamic nature of the TCP port in srv.URL
		if err := os.RemoveAll(filepath.Join(dir, ".task")); err != nil {
			t.Logf("error cleaning up: %s", err)
		}
	})

	taskfiles, err := fs.Glob(os.DirFS(dir), "root-taskfile-*.yml")
	require.NoError(t, err)

	remotes := []struct {
		name string
		root string
	}{
		{
			name: "local",
			root: ".",
		},
		{
			name: "http-remote",
			root: srv.URL,
		},
	}

	tcs := []struct {
		name, dir string
	}{
		{
			name: "second-with-dir-1:third-with-dir-1:default",
			dir:  filepath.Join(dir, "dir-1"),
		},
		{
			name: "second-with-dir-1:third-with-dir-2:default",
			dir:  filepath.Join(dir, "dir-2"),
		},
	}

	for _, taskfile := range taskfiles {
		for _, remote := range remotes { //nolint:paralleltest // sets INCLUDE_ROOT per iteration
			t.Setenv("INCLUDE_ROOT", remote.root)

			NewExecutorTest(t,
				WithName(fmt.Sprintf("%s/%s", taskfile, remote.name)),
				WithExecutorOptions(
					task.WithEntrypoint(filepath.Join(dir, taskfile)),
					task.WithDir(dir),
					task.WithInsecure(true),
					task.WithDownload(true),
					task.WithAssumeYes(true),
					task.WithVerbose(true),
					task.WithTimeout(time.Minute),
				),
				WithNoRun(),
				WithAssert(func(t *testing.T, r *ExecutorTestResult) {
					t.Helper()
					for _, tc := range tcs {
						compiled, err := r.Executor.CompiledTask(&task.Call{Task: tc.name})
						require.NoError(t, err)
						assert.Equal(t, tc.dir, compiled.Dir)
					}
				}),
			)
		}
	}
}

func TestSupportedFileNames(t *testing.T) {
	t.Parallel()

	fileNames := []string{
		"Taskfile.yml",
		"Taskfile.yaml",
		"Taskfile.dist.yml",
		"Taskfile.dist.yaml",
	}
	for _, fileName := range fileNames {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()
			NewExecutorTest(t,
				WithExecutorOptions(task.WithDir(fmt.Sprintf("testdata/file_names/%s", fileName))),
			)
		})
	}
}

func TestDynamicVariablesShouldRunOnTheTaskDir(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dir/dynamic_var")),
	)
}

func TestDotenvShouldIncludeAllEnvFiles(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv/default")),
	)
}

func TestDotenvShouldAllowMissingEnv(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv/missing_env")),
	)
}

func TestDotenvHasLocalEnvInPath(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv/local_env_in_path")),
	)
}

func TestDotenvHasLocalVarInPath(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv/local_var_in_path")),
	)
}

func TestDotenvHasEnvVarInPath(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	t.Setenv("ENV_VAR", "testing")

	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv/env_var_in_path")),
	)
}

func TestTaskDotenv(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv_task/default")),
		WithTask("dotenv"),
	)
}

func TestTaskDotenvFail(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv_task/default")),
		WithTask("no-dotenv"),
	)
}

func TestTaskDotenvOverriddenByEnv(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv_task/default")),
		WithTask("dotenv-overridden-by-env"),
	)
}

func TestTaskDotenvWithVarName(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv_task/default")),
		WithTask("dotenv-with-var-name"),
	)
}

func TestRunOnlyRunsJobsHashOnce(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/run")),
		WithTask("generate-hash"),
	)
}

func TestRunOnlyRunsJobsHashOnceWithWildcard(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/run")),
		WithTask("deploy"),
	)
}

func TestSingleCmdDep(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/single_cmd_dep")),
		WithTask("foo"),
	)
}

func TestShortTaskNotation(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/short_task_notation"),
			task.WithSilent(true),
		),
	)
}

func TestExitCodeZero(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/exit_code")),
		WithTask("exit-zero"),
	)
}

func TestExitCodeOne(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/exit_code")),
		WithTask("exit-one"),
		WithRunError(),
	)
}

func TestOutputGroup(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/output_group")),
		WithTask("bye"),
	)
}

func TestOutputGroupErrorOnlySwallowsOutputOnSuccess(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/output_group_error_only")),
		WithTask("passing"),
	)
}

func TestOutputGroupErrorOnlyShowsOutputOnFailure(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/output_group_error_only")),
		WithTask("failing"),
		WithRunError(),
	)
}

func TestIncludedVars(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/include_with_vars")),
		WithTask("task1"),
	)
}

func TestIncludedVarsMultiLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/include_with_vars_multi_level")),
	)
}

func TestTaskfileWalk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  string
	}{
		{name: "walk from root directory", dir: "testdata/taskfile_walk"},
		{name: "walk from sub directory", dir: "testdata/taskfile_walk/foo"},
		{name: "walk from sub sub directory", dir: "testdata/taskfile_walk/foo/bar"},
	}
	for _, test := range tests {
		NewExecutorTest(t,
			WithName(test.name),
			WithExecutorOptions(task.WithDir(test.dir)),
		)
	}
}

func TestPOSIXShellOptsGlobalLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/shopts/global_level")),
		WithTask("pipefail"),
	)
}

func TestPOSIXShellOptsTaskLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/shopts/task_level")),
		WithTask("pipefail"),
	)
}

func TestPOSIXShellOptsCommandLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/shopts/command_level")),
		WithTask("pipefail"),
	)
}

func TestBashShellOptsGlobalLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/shopts/global_level")),
		WithTask("globstar"),
	)
}

func TestBashShellOptsTaskLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/shopts/task_level")),
		WithTask("globstar"),
	)
}

func TestBashShellOptsCommandLevel(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/shopts/command_level")),
		WithTask("globstar"),
	)
}

func TestSplitArgs(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/split_args"),
			task.WithSilent(true),
		),
		WithVar("CLI_ARGS", "foo bar 'foo bar baz'"),
	)
}

func TestWildcard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		call    string
		wantErr bool
	}{
		{name: "basic wildcard", call: "wildcard-foo"},
		{name: "double wildcard", call: "foo-wildcard-bar"},
		{name: "store wildcard", call: "start-foo"},
		{name: "alias", call: "s-foo"},
		{name: "matches exactly", call: "matches-exactly-*"},
		{name: "no matches", call: "no-match", wantErr: true},
		{name: "multiple matches", call: "wildcard-foo-bar"},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.call),
			WithExecutorOptions(
				task.WithDir("testdata/wildcards"),
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

func TestIgnoreNilElements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  string
	}{
		{"nil cmd", "testdata/ignore_nil_elements/cmds"},
		{"nil dep", "testdata/ignore_nil_elements/deps"},
		{"nil include", "testdata/ignore_nil_elements/includes"},
		{"nil precondition", "testdata/ignore_nil_elements/preconditions"},
	}

	for _, test := range tests {
		NewExecutorTest(t,
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir(test.dir),
				task.WithSilent(true),
			),
		)
	}
}

func TestRunWhenChanged(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/run_when_changed"),
			task.WithForceAll(true),
			task.WithSilent(true),
		),
		WithTask("start"),
	)
}

func TestRunOnceSharedFailurePropagates(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/run_once_failure")),
		WithRunError(),
	)
}

func TestForce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		force    bool
		forceAll bool
	}{
		{name: "force", force: true},
		{name: "force-all", forceAll: true},
		{name: "force with gentle force experiment", force: true},
		{name: "force-all with gentle force experiment", forceAll: true},
	}
	for _, tt := range tests {
		NewExecutorTest(t,
			WithName(tt.name),
			WithExecutorOptions(
				task.WithDir("testdata/force"),
				task.WithForce(tt.force),
				task.WithForceAll(tt.forceAll),
			),
			WithTask("task-with-dep"),
		)
	}
}

func TestIncludesInternal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		task        string
		expectedErr bool
	}{
		{"included internal task via task", "task-1", false},
		{"included internal task via dep", "task-2", false},
		{"included internal direct", "included:task-3", true},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir("testdata/internal_task"),
				task.WithSilent(true),
			),
			WithTask(test.task),
		}
		if test.expectedErr {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestInternalTask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		task        string
		expectedErr bool
	}{
		{"internal task via task", "task-1", false},
		{"internal task via dep", "task-2", false},
		{"internal direct", "task-3", true},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir("testdata/internal_task"),
				task.WithSilent(true),
			),
			WithTask(test.task),
		}
		if test.expectedErr {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestIncludesInterpolation(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	const dir = "testdata/includes_interpolation"
	tests := []struct {
		name string
		task string
	}{
		{"include", "include"},
		{"include_with_env_variable", "include-with-env-variable"},
		{"include_with_dir", "include-with-dir"},
	}
	t.Setenv("MODULE", "included")

	for _, test := range tests { // nolint:paralleltest // cannot run in parallel
		NewExecutorTest(t,
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir(filepath.Join(dir, test.name)),
				task.WithSilent(true),
			),
			WithTask(test.task),
		)
	}
}

func TestIncludesFlatten(t *testing.T) {
	t.Parallel()

	const dir = "testdata/includes_flatten"
	tests := []struct {
		name        string
		taskfile    string
		task        string
		expectedErr bool
	}{
		{name: "included flatten", taskfile: "Taskfile.yml", task: "gen"},
		{name: "included flatten with default", taskfile: "Taskfile.yml", task: "default"},
		{name: "included flatten can call entrypoint tasks", taskfile: "Taskfile.yml", task: "from_entrypoint"},
		{name: "included flatten with deps", taskfile: "Taskfile.yml", task: "with_deps"},
		{name: "included flatten nested", taskfile: "Taskfile.yml", task: "from_nested"},
		{name: "included flatten multiple same task", taskfile: "Taskfile.multiple.yml", task: "gen", expectedErr: true},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir(dir),
				task.WithEntrypoint(dir+"/"+test.taskfile),
				task.WithSilent(true),
			),
			WithTask(test.task),
		}
		if test.expectedErr {
			opts = append(opts, WithSetupError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestTaskIgnoreErrors(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("task-should-pass"),
		WithExecutorOptions(task.WithDir("testdata/ignore_errors")),
		WithTask("task-should-pass"),
	)
	NewExecutorTest(t,
		WithName("task-should-fail"),
		WithExecutorOptions(task.WithDir("testdata/ignore_errors")),
		WithTask("task-should-fail"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("cmd-should-pass"),
		WithExecutorOptions(task.WithDir("testdata/ignore_errors")),
		WithTask("cmd-should-pass"),
	)
	NewExecutorTest(t,
		WithName("cmd-should-fail"),
		WithExecutorOptions(task.WithDir("testdata/ignore_errors")),
		WithTask("cmd-should-fail"),
		WithRunError(),
	)
}

func TestDeferredCmds(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("task-2"),
		WithExecutorOptions(task.WithDir("testdata/deferred")),
		WithTask("task-2"),
		WithRunError(),
	)
	NewExecutorTest(t,
		WithName("parent"),
		WithExecutorOptions(task.WithDir("testdata/deferred")),
		WithTask("parent"),
	)
}

func TestIncludeCycle(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/includes_cycle"),
			task.WithSilent(true),
		),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestIncludesIncorrect(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/includes_incorrect"),
			task.WithSilent(true),
		),
		WithSetupError(),
	)
}

func TestIncludesMissingTaskfile(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/includes_missing_taskfile"),
			task.WithSilent(true),
		),
		WithSetupError(),
	)
}

func TestIncludesOptionalImplicitFalse(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_optional_implicit_false")),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestIncludesOptionalExplicitFalse(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/includes_optional_explicit_false")),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestDotenvShouldErrorWhenIncludingDependantDotenvs(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/dotenv/error_included_envs"),
			task.WithSummary(true),
		),
		WithSetupError(),
	)
}

func TestTaskDotenvParseErrorMessage(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dotenv/parse_error")),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestDisplaysErrorOnVersion1Schema(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/version/v1"),
			task.WithVersionCheck(true),
		),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestDisplaysErrorOnVersion2Schema(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/version/v2"),
			task.WithVersionCheck(true),
		),
		WithSetupError(),
		WithFixtureTemplating(),
	)
}

func TestExpand(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/expand")),
		WithTask("pwd"),
		WithFixtureTemplateData("HOME", filepath.ToSlash(home)),
	)
}

func TestUserWorkingDirectory(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/user_working_dir")),
		WithFixtureTemplating(),
	)
}

func TestAbsPath(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/abs_path"),
			task.WithSilent(true),
		),
		WithFixtureTemplating(),
	)
}

func TestPlatforms(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/platforms")),
		WithTask("build-"+runtime.GOOS),
		WithFixtureTemplateData("GOOS", runtime.GOOS),
	)
}

func TestIncludedTaskfileVarMerging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		task string
	}{
		{"foo", "foo:pwd"},
		{"bar", "bar:pwd"},
	}
	for _, test := range tests {
		NewExecutorTest(t,
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir("testdata/included_taskfile_var_merging"),
				task.WithSilent(true),
			),
			WithTask(test.task),
			WithFixtureTemplating(),
		)
	}
}

func TestIncludesRelativePath(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("common:pwd"),
		WithExecutorOptions(task.WithDir("testdata/includes_rel_path")),
		WithTask("common:pwd"),
		WithFixtureTemplating(),
	)
	NewExecutorTest(t,
		WithName("included:common:pwd"),
		WithExecutorOptions(task.WithDir("testdata/includes_rel_path")),
		WithTask("included:common:pwd"),
		WithFixtureTemplating(),
	)
}

func TestWhenNoDirAttributeItRunsInSameDirAsTaskfile(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dir")),
		WithTask("whereami"),
		WithFixtureTemplating(),
	)
}

func TestWhenDirAttributeAndDirExistsItRunsInThatDir(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dir/explicit_exists")),
		WithTask("whereami"),
		WithFixtureTemplating(),
	)
}

func TestWhenDirAttributeItCreatesMissingAndRunsInThatDir(t *testing.T) {
	t.Parallel()

	const toBeCreated = "testdata/dir/explicit_doesnt_exist/createme"

	// Ensure that the directory to be created doesn't actually exist.
	_ = os.RemoveAll(toBeCreated)
	if _, err := os.Stat(toBeCreated); err == nil {
		t.Errorf("Directory should not exist: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(toBeCreated) })

	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dir/explicit_doesnt_exist/")),
		WithTask("whereami"),
		WithFixtureTemplating(),
	)
}

func TestDynamicVariablesRunOnTheNewCreatedDir(t *testing.T) {
	t.Parallel()

	const toBeCreated = "testdata/dir/dynamic_var_on_created_dir/created"

	// Ensure that the directory to be created doesn't actually exist.
	_ = os.RemoveAll(toBeCreated)
	if _, err := os.Stat(toBeCreated); err == nil {
		t.Errorf("Directory should not exist: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(toBeCreated) })

	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/dir/dynamic_var_on_created_dir")),
		WithFixtureTemplating(),
		// Take only the first line, as Windows may output additional debug info.
		WithPostProcessFn(PPFirstLine),
	)
}

func TestEvaluateSymlinksInPaths(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	const dir = "testdata/evaluate_symlinks_in_paths"
	t.Cleanup(func() {
		_ = os.RemoveAll(dir + "/.task")
	})

	steps := []struct {
		name string
		task string
	}{
		{"default (1)", "default"},
		{"test-sym (1)", "test-sym"},
		{"default (2)", "default"},
		{"default (3)", "default"},
		{"reset", "reset"},
	}
	for _, step := range steps { // nolint:paralleltest // cannot run in parallel
		NewExecutorTest(t,
			WithName(step.name),
			WithExecutorOptions(task.WithDir(dir)),
			WithTask(step.task),
		)
	}
}

func TestIgnoreErrorsOnTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		task        string
		expectError bool
	}{
		{name: "ignored at task level", task: "task-timeout-should-pass"},
		{name: "ignored at command level", task: "cmd-timeout-should-pass"},
		{name: "not ignored", task: "cmd-timeout-should-fail", expectError: true},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithExecutorOptions(task.WithDir("testdata/ignore_errors")),
			WithTask(test.task),
		}
		if test.expectError {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestExitImmediately(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/exit_immediately"),
			task.WithSilent(true),
		),
		WithRunError(),
	)
}

func TestRunOnceSharedDeps(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/run_once_shared_deps"),
			task.WithForceAll(true),
		),
		WithTask("build"),
		// service-a:build and service-b:build run concurrently, so their
		// output can interleave in either order, and whichever of them wins
		// the race is credited with the shared "run: once" library:build dep.
		WithPostProcessFn(func(t *testing.T, b []byte) []byte {
			t.Helper()
			re := regexp.MustCompile(`service-[ab]:library:build`)
			return re.ReplaceAll(b, []byte("service-x:library:build"))
		}),
		WithPostProcessFn(PPSortedLines),
	)
}

func TestCommandTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		task        string
		expectError bool
	}{
		{name: "timeout exceeded", task: "timeout-exceeded", expectError: true},
		{name: "timeout not exceeded", task: "timeout-not-exceeded"},
		{name: "no timeout", task: "no-timeout"},
		{name: "multiple commands with timeout", task: "multiple-cmds-timeout", expectError: true},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithExecutorOptions(task.WithDir("testdata/timeout")),
			WithTask(test.task),
		}
		if test.expectError {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestIncludesWithExclude(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		task        string
		expectError bool
	}{
		{name: "included:bar", task: "included:bar"},
		{name: "included:foo", task: "included:foo", expectError: true},
		{name: "included:foo:child", task: "included:foo:child"},
		{name: "included:namespace", task: "included:namespace"},
		{name: "included:namespace:one", task: "included:namespace:one", expectError: true},
		{name: "included:namespace-other:one", task: "included:namespace-other:one"},
		{name: "bar", task: "bar", expectError: true},
		{name: "foo", task: "foo"},
		{name: "namespace", task: "namespace"},
		{name: "namespace:two", task: "namespace:two", expectError: true},
		{name: "namespace-other:one", task: "namespace-other:one"},
	}

	for _, test := range tests {
		opts := []ExecutorTestOption{
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir("testdata/includes_with_excludes"),
				task.WithSilent(true),
			),
			WithTask(test.task),
		}
		if test.expectError {
			opts = append(opts, WithRunError())
		}
		NewExecutorTest(t, opts...)
	}
}

func TestCyclicDep(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/cyclic")),
		WithTask("task-1"),
		WithRunError(),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			var taskCalledTooManyTimesError *errors.TaskCalledTooManyTimesError
			assert.ErrorAs(t, r.Err, &taskCalledTooManyTimesError)
		}),
	)
}

func TestTaskVersion(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("v1"),
		WithExecutorOptions(
			task.WithDir("testdata/version/v1"),
			task.WithVersionCheck(true),
		),
		WithSetupError(),
		WithFixtureTemplating(),
	)
	NewExecutorTest(t,
		WithName("v2"),
		WithExecutorOptions(
			task.WithDir("testdata/version/v2"),
			task.WithVersionCheck(true),
		),
		WithSetupError(),
		WithFixtureTemplating(),
	)
	NewExecutorTest(t,
		WithName("v3"),
		WithExecutorOptions(
			task.WithDir("testdata/version/v3"),
			task.WithVersionCheck(true),
		),
		WithNoRun(),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			assert.Equal(t, semver.MustParse("3"), r.Executor.Taskfile.Version)
			assert.Equal(t, 2, r.Executor.Taskfile.Tasks.Len())
		}),
	)
}

func TestDry(t *testing.T) {
	t.Parallel()

	_ = os.Remove(filepathext.SmartJoin("testdata/dry", "file.txt"))

	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/dry"),
			task.WithDry(true),
		),
		WithTask("build"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			_, err := os.Stat(filepathext.SmartJoin(r.Executor.Dir, "file.txt"))
			assert.Error(t, err, "file.txt should not exist in dry mode")
		}),
	)
}

func TestDryChecksum(t *testing.T) {
	t.Parallel()

	const dir = "testdata/dry_checksum"
	checksumFile := filepathext.SmartJoin(dir, ".task/checksum/default")
	_ = os.Remove(checksumFile)

	NewExecutorTest(t,
		WithName("dry"),
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithDry(true),
		),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			_, err := os.Stat(checksumFile)
			require.Error(t, err, "checksum file should not exist")
		}),
	)
	NewExecutorTest(t,
		WithName("not dry"),
		WithExecutorOptions(task.WithDir(dir)),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			_, err := os.Stat(checksumFile)
			require.NoError(t, err, "checksum file should exist")
		}),
	)
}

func TestStatusVariables(t *testing.T) {
	t.Parallel()

	const dir = "testdata/status_vars"
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.Remove(filepathext.SmartJoin(dir, "generated.txt"))

	NewExecutorTest(t,
		WithName("build-checksum"),
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithVerbose(true),
		),
		WithTask("build-checksum"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			assert.Contains(t, r.Output, "3e464c4b03f4b65d740e1e130d4d108a")
		}),
	)
	NewExecutorTest(t,
		WithName("build-ts"),
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithVerbose(true),
		),
		WithTask("build-ts"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			inf, err := os.Stat(filepathext.SmartJoin(dir, "source.txt"))
			require.NoError(t, err)
			assert.Contains(t, r.Output, fmt.Sprintf("%d", inf.ModTime().Unix()))
			assert.Contains(t, r.Output, inf.ModTime().String())
		}),
	)
}

func TestCmdsVariables(t *testing.T) {
	t.Parallel()

	const dir = "testdata/cmds_vars"
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))

	NewExecutorTest(t,
		WithName("build-checksum"),
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithVerbose(true),
		),
		WithTask("build-checksum"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			assert.Contains(t, r.Output, "3e464c4b03f4b65d740e1e130d4d108a")
		}),
	)
	NewExecutorTest(t,
		WithName("build-ts"),
		WithExecutorOptions(
			task.WithDir(dir),
			task.WithVerbose(true),
		),
		WithTask("build-ts"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			inf, err := os.Stat(filepathext.SmartJoin(dir, "source.txt"))
			require.NoError(t, err)
			assert.Contains(t, r.Output, fmt.Sprintf("%d", inf.ModTime().Unix()))
			assert.Contains(t, r.Output, inf.ModTime().String())
		}),
	)
}

func TestFingerprintVarMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dir          string
		executorOpts []task.ExecutorOption
		wantErr      bool
		assertOutput func(t *testing.T, output string)
	}{
		{
			name: "TIMESTAMP is injected when the method is inherited from the Taskfile",
			dir:  "testdata/method_taskfile_timestamp",
			assertOutput: func(t *testing.T, output string) {
				t.Helper()
				// An unresolved variable renders as an empty string, so this
				// has to match an actual timestamp, not just the prefix.
				assert.Regexp(t, `ts=\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`, output)
			},
		},
		{
			name: "no variable is injected when the effective method is none",
			dir:  "testdata/method_taskfile_none",
			assertOutput: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "cs=\n")
			},
		},
		{
			name:         "an invalid method doesn't fail a run that skips fingerprinting",
			dir:          "testdata/method_invalid",
			executorOpts: []task.ExecutorOption{task.WithForce(true)},
			assertOutput: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "cs=[]\n")
			},
		},
		{
			name:    "an invalid method is still reported by the up-to-date check",
			dir:     "testdata/method_invalid",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		_ = os.RemoveAll(filepathext.SmartJoin(tt.dir, ".task"))

		opts := []ExecutorTestOption{
			WithName(tt.name),
			WithExecutorOptions(append([]task.ExecutorOption{task.WithDir(tt.dir)}, tt.executorOpts...)...),
			WithTask("build"),
		}
		if tt.wantErr {
			opts = append(opts, WithRunError())
		} else {
			opts = append(opts, WithAssert(func(t *testing.T, r *ExecutorTestResult) {
				t.Helper()
				tt.assertOutput(t, r.Output)
			}))
		}
		NewExecutorTest(t, opts...)
	}
}

func TestErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		task     string
		expected int
	}{
		{name: "direct task", task: "direct", expected: 42},
		{name: "indirect task", task: "indirect", expected: 42},
	}

	for _, test := range tests {
		NewExecutorTest(t,
			WithName(test.name),
			WithExecutorOptions(
				task.WithDir("testdata/error_code"),
				task.WithSilent(true),
			),
			WithTask(test.task),
			WithRunError(),
			WithAssert(func(t *testing.T, r *ExecutorTestResult) {
				t.Helper()
				var taskRunErr *errors.TaskRunError
				require.ErrorAs(t, r.Err, &taskRunErr)
				assert.Equal(t, test.expected, taskRunErr.TaskExitCode(), "unexpected exit code from task")
			}),
		)
	}
}

func TestRunOnceJoinerHonorsItsOwnTimeout(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/run_once_timeout")),
		WithRunError(),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			// The joiner used to wait on the shared execution alone, ignoring
			// its own timeout for as long as that execution took.
			assert.Less(t, r.Duration, 5*time.Second)

			var timeoutErr *errors.TaskTimeoutError
			require.ErrorAs(t, r.Err, &timeoutErr)
			assert.Equal(t, "joiner", timeoutErr.TaskName)
			assert.NotContains(t, r.Output, "should not be reached")
		}),
	)
}

func TestDeferredTaskTimeout(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/deferred"),
			task.WithVerbose(true),
		),
		WithTask("parent-with-timeout"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			assert.Less(t, r.Duration, 500*time.Millisecond)
			assert.Contains(t, r.Output, "parent completed")
			assert.NotContains(t, r.Output, "\ncleanup completed\n")
			assert.Contains(t, r.Output, "ignored error in deferred cmd")
		}),
	)
}

func TestExitCodeTimeout(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/exit_code")),
		WithTask("exit-timeout"),
		WithRunError(),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			var runErr *errors.TaskRunError
			require.ErrorAs(t, r.Err, &runErr)
			assert.Equal(t, errors.TimeoutExitCode, runErr.TaskExitCode())
		}),
	)
}

func TestDepTimeout(t *testing.T) {
	t.Parallel()

	NewExecutorTest(t,
		WithName("timeout exceeded"),
		WithExecutorOptions(task.WithDir("testdata/dep_timeout")),
		WithTask("timeout-exceeded"),
		WithRunError(),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			assert.Less(t, r.Duration, 5*time.Second)

			var timeoutErr *errors.TaskTimeoutError
			require.ErrorAs(t, r.Err, &timeoutErr)
			assert.Equal(t, "slow", timeoutErr.TaskName)
			assert.NotContains(t, r.Output, "should not be reached")
		}),
	)
	NewExecutorTest(t,
		WithName("timeout not exceeded"),
		WithExecutorOptions(task.WithDir("testdata/dep_timeout")),
		WithTask("timeout-not-exceeded"),
	)
}

func TestCommandTimeoutBoundsIfCondition(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/timeout")),
		WithTask("slow-if-condition"),
		WithRunError(),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			assert.Less(t, r.Duration, 5*time.Second)

			var timeoutErr *errors.TaskTimeoutError
			require.ErrorAs(t, r.Err, &timeoutErr)
			// A condition that times out fails the command, it does not skip it.
			assert.NotContains(t, r.Output, "condition was met")
		}),
	)
}

func TestCommandTimeoutAttribution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		task        string
		notContains string
	}{
		{
			name:        "a command declaring no timeout is not blamed for one",
			task:        "inherited-timeout",
			notContains: "(0s)",
		},
		{
			name:        "a command is not blamed for a timeout it never reached",
			task:        "larger-child-timeout",
			notContains: "10m",
		},
	}

	for _, test := range tests {
		NewExecutorTest(t,
			WithName(test.name),
			WithExecutorOptions(task.WithDir("testdata/timeout")),
			WithTask(test.task),
			WithRunError(),
			WithAssert(func(t *testing.T, r *ExecutorTestResult) {
				t.Helper()
				assert.Contains(t, r.Err.Error(), "command timeout exceeded (500ms)")
				assert.NotContains(t, r.Err.Error(), test.notContains)

				var timeoutErr *errors.TaskTimeoutError
				require.ErrorAs(t, r.Err, &timeoutErr)
				assert.Equal(t, test.task, timeoutErr.TaskName)

				// --watch swallows context errors; a timeout must not look like one.
				assert.False(t, errors.Is(r.Err, context.DeadlineExceeded))
			}),
		)
	}
}

func TestUserWorkingDirectoryWithIncluded(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	require.NoError(t, err)
	wd = filepath.ToSlash(filepathext.SmartJoin(wd, "testdata/user_working_dir_with_includes/somedir"))

	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/user_working_dir_with_includes"),
			task.WithUserWorkingDir(wd),
		),
		WithTask("included:echo"),
		WithFixtureTemplating(),
	)
}

func TestIncludeWithVarsInInclude(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/include_with_vars_inside_include")),
		WithNoRun(),
	)
}

func TestGitignoreTaskListFallback(t *testing.T) { //nolint:paralleltest // shares testdata/gitignore with TestGitignoreChecksum
	NewExecutorTest(t,
		WithExecutorOptions(task.WithDir("testdata/gitignore")),
		WithNoRun(),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			listed, err := r.Executor.CompiledTaskForTaskList(&task.Call{Task: "build"})
			require.NoError(t, err)
			assert.True(t, listed.ShouldUseGitignore(),
				"task list should reflect the global use_gitignore fallback")

			listedOff, err := r.Executor.CompiledTaskForTaskList(&task.Call{Task: "build-no-use_gitignore"})
			require.NoError(t, err)
			assert.False(t, listedOff.ShouldUseGitignore(),
				"explicit use_gitignore: false must be preserved in the list path")
		}),
	)
}

func TestSummary(t *testing.T) {
	t.Parallel()
	NewExecutorTest(t,
		WithExecutorOptions(
			task.WithDir("testdata/summary"),
			task.WithSummary(true),
			task.WithSilent(true),
		),
		WithTasks("task-with-summary", "other-task-with-summary"),
	)
}

func TestSilence(t *testing.T) {
	t.Parallel()

	tests := []string{
		"silent",
		"chatty",
		"task-test-silent-calls-chatty-non-silenced",
		"task-test-silent-calls-chatty-silenced",
		"task-test-chatty-calls-chatty-non-silenced",
		"task-test-chatty-calls-chatty-silenced",
		"task-test-no-cmds-calls-chatty-silenced",
		"task-test-chatty-calls-silenced-cmd",
		"task-test-is-silent-depends-on-chatty-non-silenced",
		"task-test-is-silent-depends-on-chatty-silenced",
		"task-test-is-chatty-depends-on-chatty-silenced",
	}

	for i, taskName := range tests {
		opts := []ExecutorTestOption{
			WithName(taskName),
			WithExecutorOptions(task.WithDir("testdata/silent")),
			WithTask(taskName),
		}
		if i == 0 {
			// Verify that the silent flag is in place before running anything.
			opts = append(opts, WithAssert(func(t *testing.T, r *ExecutorTestResult) {
				t.Helper()
				fetchedTask, err := r.Executor.GetTask(&task.Call{Task: "task-test-silent-calls-chatty-silenced"})
				require.NoError(t, err, "Unable to look up task task-test-silent-calls-chatty-silenced")
				require.True(t, fetchedTask.Cmds[0].Silent, "The task task-test-silent-calls-chatty-silenced should have a silent call to chatty")
			}))
		}
		NewExecutorTest(t, opts...)
	}
}

func TestGenerates(t *testing.T) {
	t.Parallel()

	const dir = "testdata/generates"
	const srcTask = "sub/src.txt"
	srcFile := filepathext.SmartJoin(dir, srcTask)

	destTasks := []string{"rel.txt", "abs.txt", "my text file.txt"}
	for _, f := range append([]string{srcTask}, destTasks...) {
		_ = os.Remove(filepathext.SmartJoin(dir, f))
	}

	for _, destTask := range destTasks {
		destFile := filepathext.SmartJoin(dir, destTask)
		NewExecutorTest(t,
			WithName(destTask+" (first run)"),
			WithExecutorOptions(task.WithDir(dir)),
			WithTask(destTask),
			WithAssert(func(t *testing.T, r *ExecutorTestResult) {
				t.Helper()
				_, err := os.Stat(srcFile)
				assert.NoError(t, err, "File should exist")
				_, err = os.Stat(destFile)
				assert.NoError(t, err, "File should exist")
			}),
		)
		NewExecutorTest(t,
			WithName(destTask+" (up to date)"),
			WithExecutorOptions(task.WithDir(dir)),
			WithTask(destTask),
		)
	}
}

func TestStatusChecksum(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	const dir = "testdata/checksum"

	tests := []struct {
		files []string
		task  string
	}{
		{[]string{"generated.txt", ".task/checksum/build"}, "build"},
		{[]string{"generated-wildcard.txt", ".task/checksum/build-wildcard"}, "build-wildcard"},
		{[]string{"generated.txt", ".task/checksum/build-with-status"}, "build-with-status"},
	}

	for _, test := range tests { // nolint:paralleltest // cannot run in parallel
		for _, f := range test.files {
			_ = os.Remove(filepathext.SmartJoin(dir, f))
		}
		checksumFile := filepathext.SmartJoin(dir, test.files[1])

		var capturedTime time.Time
		NewExecutorTest(t,
			WithName(test.task+" (first run)"),
			WithExecutorOptions(task.WithDir(dir)),
			WithTask(test.task),
			WithAssert(func(t *testing.T, r *ExecutorTestResult) {
				t.Helper()
				for _, f := range test.files {
					_, err := os.Stat(filepathext.SmartJoin(dir, f))
					require.NoError(t, err)
				}
				// Capture the modification time, so we can ensure the
				// checksum file is not regenerated when the hash hasn't
				// changed.
				s, err := os.Stat(checksumFile)
				require.NoError(t, err)
				capturedTime = s.ModTime()
			}),
		)
		NewExecutorTest(t,
			WithName(test.task+" (up to date)"),
			WithExecutorOptions(task.WithDir(dir)),
			WithTask(test.task),
			WithAssert(func(t *testing.T, r *ExecutorTestResult) {
				t.Helper()
				s, err := os.Stat(checksumFile)
				require.NoError(t, err)
				assert.Equal(t, capturedTime, s.ModTime())
			}),
		)
	}
}

// TestStatusTimestamp is a regression test for https://github.com/go-task/task/issues/1230.
// When using method: timestamp, deleting a generated file should cause the task to re-run,
// not be skipped because the timestamp file is still present.
func TestStatusTimestamp(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	const dir = "testdata/timestamp"
	generatedFile := filepathext.SmartJoin(dir, "generated.txt")

	_ = os.Remove(generatedFile)
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))

	NewExecutorTest(t,
		WithName("first run"),
		WithExecutorOptions(task.WithDir(dir)),
		WithTask("build"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			_, err := os.Stat(generatedFile)
			require.NoError(t, err, "generated.txt should exist after first run")
		}),
	)
	NewExecutorTest(t,
		WithName("up to date"),
		WithExecutorOptions(task.WithDir(dir)),
		WithTask("build"),
	)

	// Delete the generated file (simulate a clean), but leave the timestamp file.
	require.NoError(t, os.Remove(generatedFile))

	NewExecutorTest(t,
		WithName("re-run after generated file removed"),
		WithExecutorOptions(task.WithDir(dir)),
		WithTask("build"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			// This is the regression: previously the task was incorrectly
			// skipped because the timestamp file was still present.
			assert.NotContains(t, r.Output, "is up to date", "task should re-run when generated file is missing")
			_, err := os.Stat(generatedFile)
			require.NoError(t, err, "generated.txt should be recreated after third run")
		}),
	)
}

// TestStatusChecksumMissingGenerated is a regression test for https://github.com/go-task/task/issues/1230.
// When using method: checksum, deleting a generated file should cause the task to re-run,
// not be skipped because the checksum file still matches.
func TestStatusChecksumMissingGenerated(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	const dir = "testdata/checksum"
	generatedFile := filepathext.SmartJoin(dir, "generated.txt")

	_ = os.Remove(generatedFile)
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))

	NewExecutorTest(t,
		WithName("first run"),
		WithExecutorOptions(task.WithDir(dir)),
		WithTask("build"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			_, err := os.Stat(generatedFile)
			require.NoError(t, err, "generated.txt should exist after first run")
		}),
	)
	NewExecutorTest(t,
		WithName("up to date"),
		WithExecutorOptions(task.WithDir(dir)),
		WithTask("build"),
	)

	// Delete the generated file (simulate a clean), but leave the checksum file.
	require.NoError(t, os.Remove(generatedFile))

	NewExecutorTest(t,
		WithName("re-run after generated file removed"),
		WithExecutorOptions(task.WithDir(dir)),
		WithTask("build"),
		WithAssert(func(t *testing.T, r *ExecutorTestResult) {
			t.Helper()
			// This is the regression: previously the task was incorrectly
			// skipped because the checksum file still matched.
			assert.NotContains(t, r.Output, "is up to date", "task should re-run when generated file is missing")
			_, err := os.Stat(generatedFile)
			require.NoError(t, err, "generated.txt should be recreated after third run")
		}),
	)
}
