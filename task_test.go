package task_test

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/experiments"
)

func init() {
	_ = os.Setenv("NO_COLOR", "1")
}

type (
	TaskTest struct {
		name                     string
		experiments              map[*experiments.Experiment]int
		postProcessFns           []PostProcessFn
		fixtureTemplateData      map[string]any
		fixtureTemplatingEnabled bool
	}
)

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

// goldenFileName makes the file path for fixture files safe for all well-known
// operating systems. Windows in particular has a lot of restrictions the
// characters that can be used in file paths.
func goldenFileName(t *testing.T) string {
	t.Helper()
	name := t.Name()
	for _, c := range []string{` `, `<`, `>`, `:`, `"`, `/`, `\`, `|`, `?`, `*`} {
		name = strings.ReplaceAll(name, c, "-")
	}
	return name
}

// writeFixture writes a fixture file for the test. The fixture file is created
// using the [goldie.Goldie] package. The fixture file is created with the
// output of the task, after any post-process functions have been applied.
func (tt *TaskTest) writeFixture(
	t *testing.T,
	g *goldie.Goldie,
	goldenFileSuffix string,
	b []byte,
) {
	t.Helper()
	// Apply any post-process functions
	for _, fn := range tt.postProcessFns {
		b = fn(t, b)
	}
	// Write the fixture file
	goldenFileName := goldenFileName(t)
	if goldenFileSuffix != "" {
		goldenFileName += "-" + goldenFileSuffix
	}
	// Create a set of data to be made available to every test fixture
	wd, err := os.Getwd()
	require.NoError(t, err)
	if tt.fixtureTemplatingEnabled {
		fixtureTemplateData := map[string]any{
			"TEST_NAME": t.Name(),
			"TEST_DIR":  filepath.ToSlash(wd),
		}
		// If the test has additional template data, copy it into the map
		if tt.fixtureTemplateData != nil {
			maps.Copy(fixtureTemplateData, tt.fixtureTemplateData)
		}
		// Normalize output before comparison (CRLF→LF, backslash→forward slash)
		g.AssertWithTemplate(t, goldenFileName, fixtureTemplateData, normalizeOutput(b))
	} else {
		g.Assert(t, goldenFileName, b)
	}
}

// writeFixtureBuffer is a wrapper for writing the main output of the task to a
// fixture file.
func (tt *TaskTest) writeFixtureBuffer(
	t *testing.T,
	g *goldie.Goldie,
	buff bytes.Buffer,
) {
	t.Helper()
	tt.writeFixture(t, g, "", buff.Bytes())
}

// writeFixtureErrSetup is a wrapper for writing the output of an error during
// the setup phase of the task to a fixture file.
func (tt *TaskTest) writeFixtureErrSetup(
	t *testing.T,
	g *goldie.Goldie,
	err error,
) {
	t.Helper()
	tt.writeFixture(t, g, "err-setup", []byte(err.Error()))
}

//
// Functional options
//

// WithName gives the test fixture output a name. This should be used when
// running multiple tests in a single test function.
func WithName(name string) *nameTestOption {
	return &nameTestOption{name: name}
}

type nameTestOption struct {
	name string
}

func (opt *nameTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.name = opt.name
}

func (opt *nameTestOption) applyToFormatterTest(t *FormatterTest) {
	t.name = opt.name
}

// WithTask sets the name of the task to run. This should be used when the task
// to run is not the default task.
func WithTask(task string) *taskTestOption {
	return &taskTestOption{task: task}
}

type taskTestOption struct {
	task string
}

func (opt *taskTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.task = opt.task
}

func (opt *taskTestOption) applyToFormatterTest(t *FormatterTest) {
	t.task = opt.task
}

// WithVar sets a variable to be passed to the task. This can be called multiple
// times to set more than one variable.
func WithVar(key string, value any) *varTestOption {
	return &varTestOption{key: key, value: value}
}

type varTestOption struct {
	key   string
	value any
}

func (opt *varTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.vars[opt.key] = opt.value
}

func (opt *varTestOption) applyToFormatterTest(t *FormatterTest) {
	t.vars[opt.key] = opt.value
}

// WithExecutorOptions sets the [task.ExecutorOption]s to be used when creating
// a [task.Executor].
func WithExecutorOptions(executorOpts ...task.ExecutorOption) *executorOptionsTestOption {
	return &executorOptionsTestOption{executorOpts: executorOpts}
}

type executorOptionsTestOption struct {
	executorOpts []task.ExecutorOption
}

func (opt *executorOptionsTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.executorOpts = slices.Concat(t.executorOpts, opt.executorOpts)
}

func (opt *executorOptionsTestOption) applyToFormatterTest(t *FormatterTest) {
	t.executorOpts = slices.Concat(t.executorOpts, opt.executorOpts)
}

// WithExperiment sets an experiment to be enabled for the test. This can be
// called multiple times to enable more than one experiment.
func WithExperiment(experiment *experiments.Experiment, value int) *experimentTestOption {
	return &experimentTestOption{experiment: experiment, value: value}
}

type experimentTestOption struct {
	experiment *experiments.Experiment
	value      int
}

func (opt *experimentTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.experiments[opt.experiment] = opt.value
}

func (opt *experimentTestOption) applyToFormatterTest(t *FormatterTest) {
	t.experiments[opt.experiment] = opt.value
}

// WithPostProcessFn adds a [PostProcessFn] function to the test. Post-process
// functions are run on the output of the task before a fixture is created. This
// can be used to remove absolute paths, sort lines, etc. This can be called
// multiple times to add more than one post-process function.
func WithPostProcessFn(fn PostProcessFn) *postProcessFnTestOption {
	return &postProcessFnTestOption{fn: fn}
}

type postProcessFnTestOption struct {
	fn PostProcessFn
}

func (opt *postProcessFnTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.postProcessFns = append(t.postProcessFns, opt.fn)
}

func (opt *postProcessFnTestOption) applyToFormatterTest(t *FormatterTest) {
	t.postProcessFns = append(t.postProcessFns, opt.fn)
}

// WithSetupError sets the test to expect an error during the setup phase of the
// task execution. A fixture will be created with the output of any errors.
func WithSetupError() *setupErrorTestOption {
	return &setupErrorTestOption{}
}

type setupErrorTestOption struct{}

func (opt *setupErrorTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.wantSetupError = true
}

func (opt *setupErrorTestOption) applyToFormatterTest(t *FormatterTest) {
	t.wantSetupError = true
}

// WithFixtureTemplating enables templating for the golden fixture files with
// the default set of data. This is useful if the golden file is dynamic in some
// way (e.g. contains user-specific directories). To add more data, see
// WithFixtureTemplateData.
func WithFixtureTemplating() *fixtureTemplatingTestOption {
	return &fixtureTemplatingTestOption{}
}

type fixtureTemplatingTestOption struct{}

func (opt *fixtureTemplatingTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.fixtureTemplatingEnabled = true
}

func (opt *fixtureTemplatingTestOption) applyToFormatterTest(t *FormatterTest) {
	t.fixtureTemplatingEnabled = true
}

// WithFixtureTemplateData adds data to the golden fixture file templates. Keys
// given here will override any existing values. This option will also enable
// global templating, so you do not need to call WithFixtureTemplating as well.
func WithFixtureTemplateData(key string, value any) *fixtureTemplateDataTestOption {
	return &fixtureTemplateDataTestOption{key, value}
}

type fixtureTemplateDataTestOption struct {
	k string
	v any
}

func (opt *fixtureTemplateDataTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.fixtureTemplatingEnabled = true
	t.fixtureTemplateData[opt.k] = opt.v
}

func (opt *fixtureTemplateDataTestOption) applyToFormatterTest(t *FormatterTest) {
	t.fixtureTemplatingEnabled = true
	t.fixtureTemplateData[opt.k] = opt.v
}

// WithInput tells the test to create a reader with the given input. This can be
// used to simulate user input when a task requires it.
func WithInput(input string) *inputTestOption {
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
func WithRunError() *runErrorTestOption {
	return &runErrorTestOption{}
}

type runErrorTestOption struct{}

func (opt *runErrorTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.wantRunError = true
}

// WithStatusError tells the test to make an additional call to
// [task.Executor.Status] after the task has been run. A fixture will be created
// with the output of any errors.
func WithStatusError() *statusErrorTestOption {
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
func WithTasks(tasks ...string) *tasksTestOption {
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
func WithNoRun() *noRunTestOption {
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
func WithAssert(fn func(t *testing.T, r *ExecutorTestResult)) *assertTestOption {
	return &assertTestOption{fn: fn}
}

type assertTestOption struct {
	fn func(t *testing.T, r *ExecutorTestResult)
}

func (opt *assertTestOption) applyToExecutorTest(t *ExecutorTest) {
	t.assertFns = append(t.assertFns, opt.fn)
}

// WithListOptions sets the list options for the formatter.
func WithListOptions(opts task.ListOptions) *listOptionsTestOption {
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
func WithListError() *listErrorTestOption {
	return &listErrorTestOption{}
}

type listErrorTestOption struct{}

func (opt *listErrorTestOption) applyToFormatterTest(t *FormatterTest) {
	t.wantListError = true
}

//
// Post-processing functions
//

// A PostProcessFn is a function that can be applied to the output of a test
// fixture before the file is written.
type PostProcessFn func(*testing.T, []byte) []byte

// PPSortedLines sorts the lines of the output of the task. This is useful when
// the order of the output is not important, but the output is expected to be
// the same each time the task is run (e.g. when running tasks in parallel).
func PPSortedLines(t *testing.T, b []byte) []byte {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	sort.Strings(lines)
	return []byte(strings.Join(lines, "\n") + "\n")
}

// PPFirstLine keeps only the first line of the output of the task. This is
// useful when a platform (e.g. Windows) may print additional, non-deterministic
// debug info after the line we actually care about.
func PPFirstLine(t *testing.T, b []byte) []byte {
	t.Helper()
	line, _, _ := bytes.Cut(b, []byte("\n"))
	return append(line, '\n')
}

// normalizeOutput normalizes cross-platform differences for byte slice comparison:
// - Converts CRLF and CR to LF (line endings)
// - Converts backslashes to forward slashes (Windows paths)
// - Handles escaped backslashes in JSON (\\) by converting to single forward slash
func normalizeOutput(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	b = bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
	// First replace escaped backslashes (common in JSON), then single backslashes
	b = bytes.ReplaceAll(b, []byte("\\\\"), []byte("/"))
	b = bytes.ReplaceAll(b, []byte("\\"), []byte("/"))
	return b
}

// normalizePathSeparators converts backslashes to forward slashes for cross-platform path comparison.
func normalizePathSeparators(s string) string {
	return strings.ReplaceAll(s, "\\", "/")
}

// NormalizedEqual compares two byte slices after normalizing output.
// This is used as a custom goldie.EqualFn for cross-platform golden file tests.
func NormalizedEqual(actual, expected []byte) bool {
	return bytes.Equal(normalizeOutput(actual), normalizeOutput(expected))
}

func TestNormalizeOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{"CRLF to LF", []byte("line1\r\nline2\r\n"), []byte("line1\nline2\n")},
		{"CR to LF", []byte("line1\rline2\r"), []byte("line1\nline2\n")},
		{"Windows path", []byte(`D:\a\task\task`), []byte(`D:/a/task/task`)},
		{"JSON escaped backslash", []byte(`{"path":"D:\\a\\task"}`), []byte(`{"path":"D:/a/task"}`)},
		{"Mixed", []byte("D:\\a\\task\r\n"), []byte("D:/a/task\n")},
		{"Unix path unchanged", []byte("/home/user/task\n"), []byte("/home/user/task\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeOutput(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizePathSeparators(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Windows path", `D:\a\task\task`, `D:/a/task/task`},
		{"Unix path unchanged", `/home/user/task`, `/home/user/task`},
		{"Mixed separators", `C:\Users/name\file`, `C:/Users/name/file`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizePathSeparators(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
