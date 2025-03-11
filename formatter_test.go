package task_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/experiments"
	"github.com/go-task/task/v3/taskfile/ast"
)

type (
	// A FormatterTestOption is a function that configures an [FormatterTest].
	FormatterTestOption interface {
		applyToFormatterTest(*FormatterTest)
	}
	// A FormatterTest is a test wrapper around a [task.Executor] to make it
	// easy to write tests for the task formatter. See [NewFormatterTest] for
	// information on creating and running FormatterTests. These tests use
	// fixture files to assert whether the result of the output is correct. If
	// Task's behavior has been changed, the fixture files can be updated by
	// running `task gen:fixtures`.
	FormatterTest struct {
		TaskTest
		task           string
		vars           map[string]any
		executorOpts   []task.ExecutorOption
		listOptions    task.ListOptions
		wantSetupError bool
		wantListError  bool
	}
)

// NewFormatterTest sets up a new [task.Executor] with the given options and
// runs a task with the given [FormatterTestOption]s. The output of the task is
// written to a set of fixture files depending on the configuration of the test.
func NewFormatterTest(t *testing.T, opts ...FormatterTestOption) {
	t.Helper()
	tt := &FormatterTest{
		task: "default",
		vars: map[string]any{},
		TaskTest: TaskTest{
			experiments: map[*experiments.Experiment]int{},
		},
	}
	// Apply the functional options
	for _, opt := range opts {
		opt.applyToFormatterTest(tt)
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

// WithListOptions sets the list options for the formatter.
func WithListOptions(opts task.ListOptions) FormatterTestOption {
	return &listOptionsTestOption{opts}
}

type listOptionsTestOption struct {
	listOptions task.ListOptions
}

func (opt *listOptionsTestOption) applyToFormatterTest(t *FormatterTest) {
	t.listOptions = opt.listOptions
}

// WithListError tells the test to expect an error when running the formatter.
// A fixture will be created with the output of any errors.
func WithListError() FormatterTestOption {
	return &listErrorTestOption{}
}

type listErrorTestOption struct{}

func (opt *listErrorTestOption) applyToFormatterTest(t *FormatterTest) {
	t.wantListError = true
}

// Helpers

// writeFixtureErrList is a wrapper for writing the output of an error when
// running the formatter to a fixture file.
func (tt *FormatterTest) writeFixtureErrList(
	t *testing.T,
	g *goldie.Goldie,
	err error,
) {
	t.Helper()
	tt.writeFixture(t, g, "err-list", []byte(err.Error()))
}

// run is the main function for running the test. It sets up the task executor,
// runs the task, and writes the output to a fixture file.
func (tt *FormatterTest) run(t *testing.T) {
	t.Helper()
	f := func(t *testing.T) {
		t.Helper()
		var buf bytes.Buffer

		opts := append(
			tt.executorOpts,
			task.ExecutorWithStdout(&buf),
			task.ExecutorWithStderr(&buf),
		)

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

		// Run the formatter and check for errors
		if _, err := e.ListTasks(tt.listOptions); tt.wantListError {
			require.Error(t, err)
			tt.writeFixtureErrList(t, g, err)
			tt.writeFixtureBuffer(t, g, buf)
			return
		} else {
			require.NoError(t, err)
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

func TestNoLabelInList(t *testing.T) {
	t.Parallel()

	const dir = "testdata/label_list"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.ExecutorWithDir(dir),
		task.ExecutorWithStdout(&buff),
		task.ExecutorWithStderr(&buff),
	)
	require.NoError(t, e.Setup())
	if _, err := e.ListTasks(task.ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
		t.Error(err)
	}
	assert.Contains(t, buff.String(), "foo")
}

// task -al case 1: listAll list all tasks
func TestListAllShowsNoDesc(t *testing.T) {
	t.Parallel()

	const dir = "testdata/list_mixed_desc"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.ExecutorWithDir(dir),
		task.ExecutorWithStdout(&buff),
		task.ExecutorWithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	var title string
	if _, err := e.ListTasks(task.ListOptions{ListAllTasks: true}); err != nil {
		t.Error(err)
	}
	for _, title = range []string{
		"foo",
		"voo",
		"doo",
	} {
		assert.Contains(t, buff.String(), title)
	}
}

// task -al case 2: !listAll list some tasks (only those with desc)
func TestListCanListDescOnly(t *testing.T) {
	t.Parallel()

	const dir = "testdata/list_mixed_desc"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.ExecutorWithDir(dir),
		task.ExecutorWithStdout(&buff),
		task.ExecutorWithStderr(&buff),
	)
	require.NoError(t, e.Setup())
	if _, err := e.ListTasks(task.ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
		t.Error(err)
	}

	var title string
	assert.Contains(t, buff.String(), "foo")
	for _, title = range []string{
		"voo",
		"doo",
	} {
		assert.NotContains(t, buff.String(), title)
	}
}

func TestListDescInterpolation(t *testing.T) {
	t.Parallel()

	const dir = "testdata/list_desc_interpolation"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.ExecutorWithDir(dir),
		task.ExecutorWithStdout(&buff),
		task.ExecutorWithStderr(&buff),
	)
	require.NoError(t, e.Setup())
	if _, err := e.ListTasks(task.ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
		t.Error(err)
	}

	assert.Contains(t, buff.String(), "foo-var")
	assert.Contains(t, buff.String(), "bar-var")
}
