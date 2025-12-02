package task_test

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"maps"
	rand "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
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

func init() {
	_ = os.Setenv("NO_COLOR", "1")
}

type (
	TestOption interface {
		ExecutorTestOption
		FormatterTestOption
	}
	TaskTest struct {
		name                     string
		experiments              map[*experiments.Experiment]int
		postProcessFns           []PostProcessFn
		fixtureTemplateData      map[string]any
		fixtureTemplatingEnabled bool
	}
)

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
			"TEST_DIR":  wd,
		}
		// If the test has additional template data, copy it into the map
		if tt.fixtureTemplateData != nil {
			maps.Copy(fixtureTemplateData, tt.fixtureTemplateData)
		}
		g.AssertWithTemplate(t, goldenFileName, fixtureTemplateData, b)
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

// Functional options

// WithName gives the test fixture output a name. This should be used when
// running multiple tests in a single test function.
func WithName(name string) TestOption {
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
func WithTask(task string) TestOption {
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
func WithVar(key string, value any) TestOption {
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
func WithExecutorOptions(executorOpts ...task.ExecutorOption) TestOption {
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
func WithExperiment(experiment *experiments.Experiment, value int) TestOption {
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
func WithPostProcessFn(fn PostProcessFn) TestOption {
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
func WithSetupError() TestOption {
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
func WithFixtureTemplating() TestOption {
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
func WithFixtureTemplateData(key string, value any) TestOption {
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

// Post-processing

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

// fileContentTest provides a basic reusable test-case for running a Taskfile
// and inspect generated files.
type fileContentTest struct {
	Dir        string
	Entrypoint string
	Target     string
	TrimSpace  bool
	Files      map[string]string
}

func (fct fileContentTest) name(file string) string {
	return fmt.Sprintf("target=%q,file=%q", fct.Target, file)
}

func (fct fileContentTest) Run(t *testing.T) {
	t.Helper()

	for f := range fct.Files {
		_ = os.Remove(filepathext.SmartJoin(fct.Dir, f))
	}

	e := task.NewExecutor(
		task.WithDir(fct.Dir),
		task.WithTempDir(task.TempDir{
			Remote:      filepathext.SmartJoin(fct.Dir, ".task"),
			Fingerprint: filepathext.SmartJoin(fct.Dir, ".task"),
		}),
		task.WithEntrypoint(fct.Entrypoint),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
	)

	require.NoError(t, e.Setup(), "e.Setup()")
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: fct.Target}), "e.Run(target)")
	for name, expectContent := range fct.Files {
		t.Run(fct.name(name), func(t *testing.T) {
			path := filepathext.SmartJoin(e.Dir, name)
			b, err := os.ReadFile(path)
			require.NoError(t, err, "Error reading file")
			s := string(b)
			if fct.TrimSpace {
				s = strings.TrimSpace(s)
			}
			assert.Equal(t, expectContent, s, "unexpected file content in %s", path)
		})
	}
}

func TestGenerates(t *testing.T) {
	t.Parallel()

	const dir = "testdata/generates"

	const (
		srcTask        = "sub/src.txt"
		relTask        = "rel.txt"
		absTask        = "abs.txt"
		fileWithSpaces = "my text file.txt"
	)

	srcFile := filepathext.SmartJoin(dir, srcTask)

	for _, task := range []string{srcTask, relTask, absTask, fileWithSpaces} {
		path := filepathext.SmartJoin(dir, task)
		_ = os.Remove(path)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("File should not exist: %v", err)
		}
	}

	buff := bytes.NewBuffer(nil)
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(buff),
		task.WithStderr(buff),
	)
	require.NoError(t, e.Setup())

	for _, theTask := range []string{relTask, absTask, fileWithSpaces} {
		destFile := filepathext.SmartJoin(dir, theTask)
		upToDate := fmt.Sprintf("task: Task \"%s\" is up to date\n", srcTask) +
			fmt.Sprintf("task: Task \"%s\" is up to date\n", theTask)

		// Run task for the first time.
		require.NoError(t, e.Run(t.Context(), &task.Call{Task: theTask}))

		if _, err := os.Stat(srcFile); err != nil {
			t.Errorf("File should exist: %v", err)
		}
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("File should exist: %v", err)
		}
		// Ensure task was not incorrectly found to be up-to-date on first run.
		if buff.String() == upToDate {
			t.Errorf("Wrong output message: %s", buff.String())
		}
		buff.Reset()

		// Re-run task to ensure it's now found to be up-to-date.
		require.NoError(t, e.Run(t.Context(), &task.Call{Task: theTask}))
		if buff.String() != upToDate {
			t.Errorf("Wrong output message: %s", buff.String())
		}
		buff.Reset()
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
		t.Run(test.task, func(t *testing.T) {
			for _, f := range test.files {
				_ = os.Remove(filepathext.SmartJoin(dir, f))

				_, err := os.Stat(filepathext.SmartJoin(dir, f))
				require.Error(t, err)
			}

			var buff bytes.Buffer
			tempDir := task.TempDir{
				Remote:      filepathext.SmartJoin(dir, ".task"),
				Fingerprint: filepathext.SmartJoin(dir, ".task"),
			}
			e := task.NewExecutor(
				task.WithDir(dir),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithTempDir(tempDir),
			)
			require.NoError(t, e.Setup())

			require.NoError(t, e.Run(t.Context(), &task.Call{Task: test.task}))
			for _, f := range test.files {
				_, err := os.Stat(filepathext.SmartJoin(dir, f))
				require.NoError(t, err)
			}

			// Capture the modification time, so we can ensure the checksum file
			// is not regenerated when the hash hasn't changed.
			s, err := os.Stat(filepathext.SmartJoin(tempDir.Fingerprint, "checksum/"+test.task))
			require.NoError(t, err)
			time := s.ModTime()

			buff.Reset()
			require.NoError(t, e.Run(t.Context(), &task.Call{Task: test.task}))
			assert.Equal(t, `task: Task "`+test.task+`" is up to date`+"\n", buff.String())

			s, err = os.Stat(filepathext.SmartJoin(tempDir.Fingerprint, "checksum/"+test.task))
			require.NoError(t, err)
			assert.Equal(t, time, s.ModTime())
		})
	}
}

func TestStatusVariables(t *testing.T) {
	t.Parallel()

	const dir = "testdata/status_vars"

	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.Remove(filepathext.SmartJoin(dir, "generated.txt"))

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithTempDir(task.TempDir{
			Remote:      filepathext.SmartJoin(dir, ".task"),
			Fingerprint: filepathext.SmartJoin(dir, ".task"),
		}),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(false),
		task.WithVerbose(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build-checksum"}))

	assert.Contains(t, buff.String(), "3e464c4b03f4b65d740e1e130d4d108a")

	buff.Reset()
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build-ts"}))

	inf, err := os.Stat(filepathext.SmartJoin(dir, "source.txt"))
	require.NoError(t, err)
	ts := fmt.Sprintf("%d", inf.ModTime().Unix())
	tf := inf.ModTime().String()

	assert.Contains(t, buff.String(), ts)
	assert.Contains(t, buff.String(), tf)
}

func TestCmdsVariables(t *testing.T) {
	t.Parallel()

	const dir = "testdata/cmds_vars"

	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithTempDir(task.TempDir{
			Remote:      filepathext.SmartJoin(dir, ".task"),
			Fingerprint: filepathext.SmartJoin(dir, ".task"),
		}),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(false),
		task.WithVerbose(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build-checksum"}))

	assert.Contains(t, buff.String(), "3e464c4b03f4b65d740e1e130d4d108a")

	buff.Reset()
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build-ts"}))
	inf, err := os.Stat(filepathext.SmartJoin(dir, "source.txt"))
	require.NoError(t, err)
	ts := fmt.Sprintf("%d", inf.ModTime().Unix())
	tf := inf.ModTime().String()

	assert.Contains(t, buff.String(), ts)
	assert.Contains(t, buff.String(), tf)
}

func TestCyclicDep(t *testing.T) {
	t.Parallel()

	const dir = "testdata/cyclic"

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
	)
	require.NoError(t, e.Setup())
	err := e.Run(t.Context(), &task.Call{Task: "task-1"})
	var taskCalledTooManyTimesError *errors.TaskCalledTooManyTimesError
	assert.ErrorAs(t, err, &taskCalledTooManyTimesError)
}

func TestTaskVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Dir     string
		Version *semver.Version
		wantErr bool
	}{
		{"testdata/version/v1", semver.MustParse("1"), true},
		{"testdata/version/v2", semver.MustParse("2"), true},
		{"testdata/version/v3", semver.MustParse("3"), false},
	}

	for _, test := range tests {
		t.Run(test.Dir, func(t *testing.T) {
			t.Parallel()

			e := task.NewExecutor(
				task.WithDir(test.Dir),
				task.WithStdout(io.Discard),
				task.WithStderr(io.Discard),
				task.WithVersionCheck(true),
			)
			err := e.Setup()
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.Version, e.Taskfile.Version)
			assert.Equal(t, 2, e.Taskfile.Tasks.Len())
		})
	}
}

func TestTaskIgnoreErrors(t *testing.T) {
	t.Parallel()

	const dir = "testdata/ignore_errors"

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
	)
	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "task-should-pass"}))
	require.Error(t, e.Run(t.Context(), &task.Call{Task: "task-should-fail"}))
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "cmd-should-pass"}))
	require.Error(t, e.Run(t.Context(), &task.Call{Task: "cmd-should-fail"}))
}

func TestExpand(t *testing.T) {
	t.Parallel()

	const dir = "testdata/expand"

	home, err := os.UserHomeDir()
	if err != nil {
		t.Errorf("Couldn't get $HOME: %v", err)
	}
	var buff bytes.Buffer

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "pwd"}))
	assert.Equal(t, home, strings.TrimSpace(buff.String()))
}

func TestDry(t *testing.T) {
	t.Parallel()

	const dir = "testdata/dry"

	file := filepathext.SmartJoin(dir, "file.txt")
	_ = os.Remove(file)

	var buff bytes.Buffer

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithDry(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))

	assert.Equal(t, "task: [build] touch file.txt", strings.TrimSpace(buff.String()))
	if _, err := os.Stat(file); err == nil {
		t.Errorf("File should not exist %s", file)
	}
}

// TestDryChecksum tests if the checksum file is not being written to disk
// if the dry mode is enabled.
func TestDryChecksum(t *testing.T) {
	t.Parallel()

	const dir = "testdata/dry_checksum"

	checksumFile := filepathext.SmartJoin(dir, ".task/checksum/default")
	_ = os.Remove(checksumFile)

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithTempDir(task.TempDir{
			Remote:      filepathext.SmartJoin(dir, ".task"),
			Fingerprint: filepathext.SmartJoin(dir, ".task"),
		}),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithDry(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))

	_, err := os.Stat(checksumFile)
	require.Error(t, err, "checksum file should not exist")

	e.Dry = false
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	_, err = os.Stat(checksumFile)
	require.NoError(t, err, "checksum file should exist")
}

func TestIncludes(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"main.txt":                                  "main",
			"included_directory.txt":                    "included_directory",
			"included_directory_without_dir.txt":        "included_directory_without_dir",
			"included_taskfile_without_dir.txt":         "included_taskfile_without_dir",
			"./module2/included_directory_with_dir.txt": "included_directory_with_dir",
			"./module2/included_taskfile_with_dir.txt":  "included_taskfile_with_dir",
			"os_include.txt":                            "os",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesMultiLevel(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes_multi_level",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"called_one.txt":   "one",
			"called_two.txt":   "two",
			"called_three.txt": "three",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesRemote(t *testing.T) {
	enableExperimentForTest(t, &experiments.RemoteTaskfiles, 1)

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
							expectedContent := fmt.Sprint(rand.Int64())
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

func TestIncludeCycle(t *testing.T) {
	t.Parallel()

	const dir = "testdata/includes_cycle"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(true),
	)

	err := e.Setup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task: include cycle detected between")
}

func TestIncludesIncorrect(t *testing.T) {
	t.Parallel()

	const dir = "testdata/includes_incorrect"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(true),
	)

	err := e.Setup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to parse testdata/includes_incorrect/incomplete.yml:", err.Error())
}

func TestIncludesEmptyMain(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes_empty",
		Target:    "included:default",
		TrimSpace: true,
		Files: map[string]string{
			"file.txt": "default",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesHttp(t *testing.T) {
	enableExperimentForTest(t, &experiments.RemoteTaskfiles, 1)

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

	for _, taskfile := range taskfiles {
		t.Run(taskfile, func(t *testing.T) {
			for _, remote := range remotes {
				t.Run(remote.name, func(t *testing.T) {
					t.Setenv("INCLUDE_ROOT", remote.root)
					entrypoint := filepath.Join(dir, taskfile)

					var buff SyncBuffer
					e := task.NewExecutor(
						task.WithEntrypoint(entrypoint),
						task.WithDir(dir),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithInsecure(true),
						task.WithDownload(true),
						task.WithAssumeYes(true),
						task.WithStdout(&buff),
						task.WithStderr(&buff),
						task.WithVerbose(true),
						task.WithTimeout(time.Minute),
					)
					require.NoError(t, e.Setup())
					defer func() { t.Log("output:", buff.buf.String()) }()

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

					for _, tc := range tcs {
						t.Run(tc.name, func(t *testing.T) {
							t.Parallel()
							task, err := e.CompiledTask(&task.Call{Task: tc.name})
							require.NoError(t, err)
							assert.Equal(t, tc.dir, task.Dir)
						})
					}
				})
			}
		})
	}
}

func TestIncludesDependencies(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes_deps",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"default.txt":     "default",
			"called_dep.txt":  "called_dep",
			"called_task.txt": "called_task",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesCallingRoot(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes_call_root_task",
		Target:    "included:call-root",
		TrimSpace: true,
		Files: map[string]string{
			"root_task.txt": "root task",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesOptional(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes_optional",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"called_dep.txt": "called_dep",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesOptionalImplicitFalse(t *testing.T) {
	t.Parallel()

	const dir = "testdata/includes_optional_implicit_false"
	wd, _ := os.Getwd()

	message := "task: No Taskfile found at \"%s/%s/TaskfileOptional.yml\""
	expected := fmt.Sprintf(message, wd, dir)

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
	)

	err := e.Setup()
	require.Error(t, err)
	assert.Equal(t, expected, err.Error())
}

func TestIncludesOptionalExplicitFalse(t *testing.T) {
	t.Parallel()

	const dir = "testdata/includes_optional_explicit_false"
	wd, _ := os.Getwd()

	message := "task: No Taskfile found at \"%s/%s/TaskfileOptional.yml\""
	expected := fmt.Sprintf(message, wd, dir)

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
	)

	err := e.Setup()
	require.Error(t, err)
	assert.Equal(t, expected, err.Error())
}

func TestIncludesFromCustomTaskfile(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Entrypoint: "testdata/includes_yaml/Custom.ext",
		Dir:        "testdata/includes_yaml",
		Target:     "default",
		TrimSpace:  true,
		Files: map[string]string{
			"main.txt":                         "main",
			"included_with_yaml_extension.txt": "included_with_yaml_extension",
			"included_with_custom_file.txt":    "included_with_custom_file",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesRelativePath(t *testing.T) {
	t.Parallel()

	const dir = "testdata/includes_rel_path"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)

	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "common:pwd"}))
	assert.Contains(t, buff.String(), "testdata/includes_rel_path/common")

	buff.Reset()
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "included:common:pwd"}))
	assert.Contains(t, buff.String(), "testdata/includes_rel_path/common")
}

func TestIncludesInternal(t *testing.T) {
	t.Parallel()

	const dir = "testdata/internal_task"
	tests := []struct {
		name           string
		task           string
		expectedErr    bool
		expectedOutput string
	}{
		{"included internal task via task", "task-1", false, "Hello, World!\n"},
		{"included internal task via dep", "task-2", false, "Hello, World!\n"},
		{"included internal direct", "included:task-3", true, "task: No tasks with description available. Try --list-all to list all tasks\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(dir),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
			)
			require.NoError(t, e.Setup())

			err := e.Run(t.Context(), &task.Call{Task: test.task})
			if test.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

func TestIncludesFlatten(t *testing.T) {
	t.Parallel()

	const dir = "testdata/includes_flatten"
	tests := []struct {
		name           string
		taskfile       string
		task           string
		expectedErr    bool
		expectedOutput string
	}{
		{name: "included flatten", taskfile: "Taskfile.yml", task: "gen", expectedOutput: "gen from included\n"},
		{name: "included flatten with default", taskfile: "Taskfile.yml", task: "default", expectedOutput: "default from included flatten\n"},
		{name: "included flatten can call entrypoint tasks", taskfile: "Taskfile.yml", task: "from_entrypoint", expectedOutput: "from entrypoint\n"},
		{name: "included flatten with deps", taskfile: "Taskfile.yml", task: "with_deps", expectedOutput: "gen from included\nwith_deps from included\n"},
		{name: "included flatten nested", taskfile: "Taskfile.yml", task: "from_nested", expectedOutput: "from nested\n"},
		{name: "included flatten multiple same task", taskfile: "Taskfile.multiple.yml", task: "gen", expectedErr: true, expectedOutput: "task: Found multiple tasks (gen) included by \"included\"\""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(dir),
				task.WithEntrypoint(dir+"/"+test.taskfile),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
			)
			err := e.Setup()
			if test.expectedErr {
				assert.EqualError(t, err, test.expectedOutput)
			} else {
				require.NoError(t, err)
				_ = e.Run(t.Context(), &task.Call{Task: test.task})
				assert.Equal(t, test.expectedOutput, buff.String())
			}
		})
	}
}

func TestIncludesInterpolation(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	const dir = "testdata/includes_interpolation"
	tests := []struct {
		name           string
		task           string
		expectedErr    bool
		expectedOutput string
	}{
		{"include", "include", false, "include\n"},
		{"include_with_env_variable", "include-with-env-variable", false, "include_with_env_variable\n"},
		{"include_with_dir", "include-with-dir", false, "included\n"},
	}
	t.Setenv("MODULE", "included")

	for _, test := range tests { // nolint:paralleltest // cannot run in parallel
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(filepath.Join(dir, test.name)),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
			)
			require.NoError(t, e.Setup())

			err := e.Run(t.Context(), &task.Call{Task: test.task})
			if test.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

func TestIncludesWithExclude(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/includes_with_excludes"),
		task.WithSilent(true),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	err := e.Run(t.Context(), &task.Call{Task: "included:bar"})
	require.NoError(t, err)
	assert.Equal(t, "bar\n", buff.String())
	buff.Reset()

	err = e.Run(t.Context(), &task.Call{Task: "included:foo"})
	require.Error(t, err)
	buff.Reset()

	err = e.Run(t.Context(), &task.Call{Task: "bar"})
	require.Error(t, err)
	buff.Reset()

	err = e.Run(t.Context(), &task.Call{Task: "foo"})
	require.NoError(t, err)
	assert.Equal(t, "foo\n", buff.String())
}

func TestIncludedTaskfileVarMerging(t *testing.T) {
	t.Parallel()

	const dir = "testdata/included_taskfile_var_merging"
	tests := []struct {
		name           string
		task           string
		expectedOutput string
	}{
		{"foo", "foo:pwd", "included_taskfile_var_merging/foo\n"},
		{"bar", "bar:pwd", "included_taskfile_var_merging/bar\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(dir),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
			)
			require.NoError(t, e.Setup())

			err := e.Run(t.Context(), &task.Call{Task: test.task})
			require.NoError(t, err)
			assert.Contains(t, buff.String(), test.expectedOutput)
		})
	}
}

func TestInternalTask(t *testing.T) {
	t.Parallel()

	const dir = "testdata/internal_task"
	tests := []struct {
		name           string
		task           string
		expectedErr    bool
		expectedOutput string
	}{
		{"internal task via task", "task-1", false, "Hello, World!\n"},
		{"internal task via dep", "task-2", false, "Hello, World!\n"},
		{"internal direct", "task-3", true, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(dir),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
			)
			require.NoError(t, e.Setup())

			err := e.Run(t.Context(), &task.Call{Task: test.task})
			if test.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

func TestIncludesShadowedDefault(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes_shadowed_default",
		Target:    "included",
		TrimSpace: true,
		Files: map[string]string{
			"file.txt": "shadowed",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestIncludesUnshadowedDefault(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/includes_unshadowed_default",
		Target:    "included",
		TrimSpace: true,
		Files: map[string]string{
			"file.txt": "included",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
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

			tt := fileContentTest{
				Dir:       fmt.Sprintf("testdata/file_names/%s", fileName),
				Target:    "default",
				TrimSpace: true,
				Files: map[string]string{
					"output.txt": "hello",
				},
			}
			tt.Run(t)
		})
	}
}

func TestSummary(t *testing.T) {
	t.Parallel()

	const dir = "testdata/summary"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSummary(true),
		task.WithSilent(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "task-with-summary"}, &task.Call{Task: "other-task-with-summary"}))

	data, err := os.ReadFile(filepathext.SmartJoin(dir, "task-with-summary.txt"))
	require.NoError(t, err)

	expectedOutput := string(data)
	if runtime.GOOS == "windows" {
		expectedOutput = strings.ReplaceAll(expectedOutput, "\r\n", "\n")
	}

	assert.Equal(t, expectedOutput, buff.String())
}

func TestWhenNoDirAttributeItRunsInSameDirAsTaskfile(t *testing.T) {
	t.Parallel()

	const expected = "dir"
	const dir = "testdata/" + expected
	var out bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&out),
		task.WithStderr(&out),
	)

	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "whereami"}))

	// got should be the "dir" part of "testdata/dir"
	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")
}

func TestWhenDirAttributeAndDirExistsItRunsInThatDir(t *testing.T) {
	t.Parallel()

	const expected = "exists"
	const dir = "testdata/dir/explicit_exists"
	var out bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&out),
		task.WithStderr(&out),
	)

	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "whereami"}))

	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")
}

func TestWhenDirAttributeItCreatesMissingAndRunsInThatDir(t *testing.T) {
	t.Parallel()

	const expected = "createme"
	const dir = "testdata/dir/explicit_doesnt_exist/"
	const toBeCreated = dir + expected
	const target = "whereami"
	var out bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&out),
		task.WithStderr(&out),
	)

	// Ensure that the directory to be created doesn't actually exist.
	_ = os.RemoveAll(toBeCreated)
	if _, err := os.Stat(toBeCreated); err == nil {
		t.Errorf("Directory should not exist: %v", err)
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: target}))

	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")

	// Clean-up after ourselves only if no error.
	_ = os.RemoveAll(toBeCreated)
}

func TestDynamicVariablesRunOnTheNewCreatedDir(t *testing.T) {
	t.Parallel()

	const expected = "created"
	const dir = "testdata/dir/dynamic_var_on_created_dir/"
	const toBeCreated = dir + expected
	const target = "default"
	var out bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&out),
		task.WithStderr(&out),
	)

	// Ensure that the directory to be created doesn't actually exist.
	_ = os.RemoveAll(toBeCreated)
	if _, err := os.Stat(toBeCreated); err == nil {
		t.Errorf("Directory should not exist: %v", err)
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: target}))

	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")

	// Clean-up after ourselves only if no error.
	_ = os.RemoveAll(toBeCreated)
}

func TestDynamicVariablesShouldRunOnTheTaskDir(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dir/dynamic_var",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"subdirectory/from_root_taskfile.txt":          "subdirectory\n",
			"subdirectory/from_included_taskfile.txt":      "subdirectory\n",
			"subdirectory/from_included_taskfile_task.txt": "subdirectory\n",
			"subdirectory/from_interpolated_dir.txt":       "subdirectory\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestDisplaysErrorOnVersion1Schema(t *testing.T) {
	t.Parallel()

	e := task.NewExecutor(
		task.WithDir("testdata/version/v1"),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithVersionCheck(true),
	)
	err := e.Setup()
	require.Error(t, err)
	assert.Regexp(t, regexp.MustCompile(`task: Invalid schema version in Taskfile \".*testdata\/version\/v1\/Taskfile\.yml\":\nSchema version \(1\.0\.0\) no longer supported\. Please use v3 or above`), err.Error())
}

func TestDisplaysErrorOnVersion2Schema(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/version/v2"),
		task.WithStdout(io.Discard),
		task.WithStderr(&buff),
		task.WithVersionCheck(true),
	)
	err := e.Setup()
	require.Error(t, err)
	assert.Regexp(t, regexp.MustCompile(`task: Invalid schema version in Taskfile \".*testdata\/version\/v2\/Taskfile\.yml\":\nSchema version \(2\.0\.0\) no longer supported\. Please use v3 or above`), err.Error())
}

func TestShortTaskNotation(t *testing.T) {
	t.Parallel()

	const dir = "testdata/short_task_notation"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	assert.Equal(t, "string-slice-1\nstring-slice-2\nstring\n", buff.String())
}

func TestDotenvShouldIncludeAllEnvFiles(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv/default",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"include.txt": "INCLUDE1='from_include1' INCLUDE2='from_include2'\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestDotenvShouldErrorWhenIncludingDependantDotenvs(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/dotenv/error_included_envs"),
		task.WithSummary(true),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)

	err := e.Setup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "move the dotenv")
}

func TestDotenvShouldAllowMissingEnv(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv/missing_env",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"include.txt": "INCLUDE1='' INCLUDE2=''\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestDotenvHasLocalEnvInPath(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv/local_env_in_path",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"var.txt": "VAR='var_in_dot_env_1'\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestDotenvHasLocalVarInPath(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv/local_var_in_path",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"var.txt": "VAR='var_in_dot_env_3'\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestDotenvHasEnvVarInPath(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	t.Setenv("ENV_VAR", "testing")

	tt := fileContentTest{
		Dir:       "testdata/dotenv/env_var_in_path",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"var.txt": "VAR='var_in_dot_env_2'\n",
		},
	}
	tt.Run(t)
}

func TestTaskDotenvParseErrorMessage(t *testing.T) {
	t.Parallel()

	e := task.NewExecutor(
		task.WithDir("testdata/dotenv/parse_error"),
	)

	path, _ := filepath.Abs(filepath.Join(e.Dir, ".env-with-error"))
	expected := fmt.Sprintf("error reading env file %s:", path)

	err := e.Setup()
	require.ErrorContains(t, err, expected)
}

func TestTaskDotenv(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "dotenv",
		TrimSpace: true,
		Files: map[string]string{
			"dotenv.txt": "foo",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestTaskDotenvFail(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "no-dotenv",
		TrimSpace: true,
		Files: map[string]string{
			"no-dotenv.txt": "global",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestTaskDotenvOverriddenByEnv(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "dotenv-overridden-by-env",
		TrimSpace: true,
		Files: map[string]string{
			"dotenv-overridden-by-env.txt": "overridden",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestTaskDotenvWithVarName(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "dotenv-with-var-name",
		TrimSpace: true,
		Files: map[string]string{
			"dotenv-with-var-name.txt": "foo",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestExitImmediately(t *testing.T) {
	t.Parallel()

	const dir = "testdata/exit_immediately"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(true),
	)
	require.NoError(t, e.Setup())

	require.Error(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	assert.Contains(t, buff.String(), `"this_should_fail": executable file not found in $PATH`)
}

func TestRunOnlyRunsJobsHashOnce(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:    "testdata/run",
		Target: "generate-hash",
		Files: map[string]string{
			"hash.txt": "starting 1\n1\n2\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestRunOnlyRunsJobsHashOnceWithWildcard(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:    "testdata/run",
		Target: "deploy",
		Files: map[string]string{
			"wildcard.txt": "Deploy infra\nDeploy js\nDeploy go\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestRunOnceSharedDeps(t *testing.T) {
	t.Parallel()

	const dir = "testdata/run_once_shared_deps"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithForceAll(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))

	rx := regexp.MustCompile(`task: \[service-[a,b]:library:build\] echo "build library"`)
	matches := rx.FindAllStringSubmatch(buff.String(), -1)
	assert.Len(t, matches, 1)
	assert.Contains(t, buff.String(), `task: [service-a:build] echo "build a"`)
	assert.Contains(t, buff.String(), `task: [service-b:build] echo "build b"`)
}

func TestRunWhenChanged(t *testing.T) {
	t.Parallel()

	const dir = "testdata/run_when_changed"

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithForceAll(true),
		task.WithSilent(true),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "start"}))
	expectedOutputOrder := strings.TrimSpace(`
login server=fubar user=fubar
login server=foo user=foo
login server=bar user=bar
`)
	assert.Contains(t, buff.String(), expectedOutputOrder)
}

func TestDeferredCmds(t *testing.T) {
	t.Parallel()

	const dir = "testdata/deferred"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [task-2] echo 'cmd ran'
cmd ran
task: [task-2] exit 1
task: [task-2] echo 'failing' && exit 2
failing
echo ran
task-1 ran successfully
task: [task-1] echo 'task-1 ran successfully'
task-1 ran successfully
`)
	require.Error(t, e.Run(t.Context(), &task.Call{Task: "task-2"}))
	assert.Contains(t, buff.String(), expectedOutputOrder)
	buff.Reset()
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "parent"}))
	assert.Contains(t, buff.String(), "child task deferred value-from-parent")
}

func TestExitCodeZero(t *testing.T) {
	t.Parallel()

	const dir = "testdata/exit_code"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "exit-zero"}))
	assert.Equal(t, "FOO=bar - DYNAMIC_FOO=bar - EXIT_CODE=", strings.TrimSpace(buff.String()))
}

func TestExitCodeOne(t *testing.T) {
	t.Parallel()

	const dir = "testdata/exit_code"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	require.Error(t, e.Run(t.Context(), &task.Call{Task: "exit-one"}))
	assert.Equal(t, "FOO=bar - DYNAMIC_FOO=bar - EXIT_CODE=1", strings.TrimSpace(buff.String()))
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
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(test.dir),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
			)
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
			assert.Equal(t, "string-slice-1\n", buff.String())
		})
	}
}

func TestOutputGroup(t *testing.T) {
	t.Parallel()

	const dir = "testdata/output_group"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [hello] echo 'Hello!'
::group::hello
Hello!
::endgroup::
task: [bye] echo 'Bye!'
::group::bye
Bye!
::endgroup::
`)
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "bye"}))
	t.Log(buff.String())
	assert.Equal(t, strings.TrimSpace(buff.String()), expectedOutputOrder)
}

func TestOutputGroupErrorOnlySwallowsOutputOnSuccess(t *testing.T) {
	t.Parallel()

	const dir = "testdata/output_group_error_only"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "passing"}))
	t.Log(buff.String())
	assert.Empty(t, buff.String())
}

func TestOutputGroupErrorOnlyShowsOutputOnFailure(t *testing.T) {
	t.Parallel()

	const dir = "testdata/output_group_error_only"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	require.Error(t, e.Run(t.Context(), &task.Call{Task: "failing"}))
	t.Log(buff.String())
	assert.Contains(t, "failing-output", strings.TrimSpace(buff.String()))
	assert.NotContains(t, "passing", strings.TrimSpace(buff.String()))
}

func TestIncludedVars(t *testing.T) {
	t.Parallel()

	const dir = "testdata/include_with_vars"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [included1:task1] echo "VAR_1 is included1-var1"
VAR_1 is included1-var1
task: [included1:task1] echo "VAR_2 is included-default-var2"
VAR_2 is included-default-var2
task: [included2:task1] echo "VAR_1 is included2-var1"
VAR_1 is included2-var1
task: [included2:task1] echo "VAR_2 is included-default-var2"
VAR_2 is included-default-var2
task: [included3:task1] echo "VAR_1 is included-default-var1"
VAR_1 is included-default-var1
task: [included3:task1] echo "VAR_2 is included-default-var2"
VAR_2 is included-default-var2
`)
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "task1"}))
	t.Log(buff.String())
	assert.Equal(t, strings.TrimSpace(buff.String()), expectedOutputOrder)
}

func TestIncludeWithVarsInInclude(t *testing.T) {
	t.Parallel()

	const dir = "testdata/include_with_vars_inside_include"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
}

func TestIncludedVarsMultiLevel(t *testing.T) {
	t.Parallel()

	const dir = "testdata/include_with_vars_multi_level"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [lib:greet] echo 'Hello world'
Hello world
task: [foo:lib:greet] echo 'Hello foo'
Hello foo
task: [bar:lib:greet] echo 'Hello bar'
Hello bar
`)
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	t.Log(buff.String())
	assert.Equal(t, expectedOutputOrder, strings.TrimSpace(buff.String()))
}

func TestErrorCode(t *testing.T) {
	t.Parallel()

	const dir = "testdata/error_code"
	tests := []struct {
		name     string
		task     string
		expected int
	}{
		{
			name:     "direct task",
			task:     "direct",
			expected: 42,
		}, {
			name:     "indirect task",
			task:     "indirect",
			expected: 42,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(dir),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
			)
			require.NoError(t, e.Setup())

			err := e.Run(t.Context(), &task.Call{Task: test.task})
			require.Error(t, err)
			taskRunErr, ok := err.(*errors.TaskRunError)
			assert.True(t, ok, "cannot cast returned error to *task.TaskRunError")
			assert.Equal(t, test.expected, taskRunErr.TaskExitCode(), "unexpected exit code from task")
		})
	}
}

func TestEvaluateSymlinksInPaths(t *testing.T) { // nolint:paralleltest // cannot run in parallel
	const dir = "testdata/evaluate_symlinks_in_paths"
	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(false),
	)
	tests := []struct {
		name     string
		task     string
		expected string
	}{
		{
			name:     "default (1)",
			task:     "default",
			expected: "task: [default] echo \"some job\"\nsome job",
		},
		{
			name:     "test-sym (1)",
			task:     "test-sym",
			expected: "task: [test-sym] echo \"shared file source changed\" > src/shared/b",
		},
		{
			name:     "default (2)",
			task:     "default",
			expected: "task: [default] echo \"some job\"\nsome job",
		},
		{
			name:     "default (3)",
			task:     "default",
			expected: `task: Task "default" is up to date`,
		},
		{
			name:     "reset",
			task:     "reset",
			expected: "task: [reset] echo \"shared file source\" > src/shared/b\ntask: [reset] echo \"file source\" > src/a",
		},
	}
	for _, test := range tests { // nolint:paralleltest // cannot run in parallel
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, e.Setup())
			err := e.Run(t.Context(), &task.Call{Task: test.task})
			require.NoError(t, err)
			assert.Equal(t, test.expected, strings.TrimSpace(buff.String()))
			buff.Reset()
		})
	}
	err := os.RemoveAll(dir + "/.task")
	require.NoError(t, err)
}

func TestTaskfileWalk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{
			name:     "walk from root directory",
			dir:      "testdata/taskfile_walk",
			expected: "foo\n",
		}, {
			name:     "walk from sub directory",
			dir:      "testdata/taskfile_walk/foo",
			expected: "foo\n",
		}, {
			name:     "walk from sub sub directory",
			dir:      "testdata/taskfile_walk/foo/bar",
			expected: "foo\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(test.dir),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
			)
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
			assert.Equal(t, test.expected, buff.String())
		})
	}
}

func TestUserWorkingDirectory(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/user_working_dir"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	assert.Equal(t, fmt.Sprintf("%s\n", wd), buff.String())
}

func TestUserWorkingDirectoryWithIncluded(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	require.NoError(t, err)

	wd = filepathext.SmartJoin(wd, "testdata/user_working_dir_with_includes/somedir")

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/user_working_dir_with_includes"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	e.UserWorkingDir = wd

	require.NoError(t, err)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "included:echo"}))
	assert.Equal(t, fmt.Sprintf("%s\n", wd), buff.String())
}

func TestPlatforms(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/platforms"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build-" + runtime.GOOS}))
	assert.Equal(t, fmt.Sprintf("task: [build-%s] echo 'Running task on %s'\nRunning task on %s\n", runtime.GOOS, runtime.GOOS, runtime.GOOS), buff.String())
}

func TestPOSIXShellOptsGlobalLevel(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/shopts/global_level"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	err := e.Run(t.Context(), &task.Call{Task: "pipefail"})
	require.NoError(t, err)
	assert.Equal(t, "pipefail\ton\n", buff.String())
}

func TestPOSIXShellOptsTaskLevel(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/shopts/task_level"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	err := e.Run(t.Context(), &task.Call{Task: "pipefail"})
	require.NoError(t, err)
	assert.Equal(t, "pipefail\ton\n", buff.String())
}

func TestPOSIXShellOptsCommandLevel(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/shopts/command_level"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	err := e.Run(t.Context(), &task.Call{Task: "pipefail"})
	require.NoError(t, err)
	assert.Equal(t, "pipefail\ton\n", buff.String())
}

func TestBashShellOptsGlobalLevel(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/shopts/global_level"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	err := e.Run(t.Context(), &task.Call{Task: "globstar"})
	require.NoError(t, err)
	assert.Equal(t, "globstar\ton\n", buff.String())
}

func TestBashShellOptsTaskLevel(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/shopts/task_level"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	err := e.Run(t.Context(), &task.Call{Task: "globstar"})
	require.NoError(t, err)
	assert.Equal(t, "globstar\ton\n", buff.String())
}

func TestBashShellOptsCommandLevel(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/shopts/command_level"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
	)
	require.NoError(t, e.Setup())

	err := e.Run(t.Context(), &task.Call{Task: "globstar"})
	require.NoError(t, err)
	assert.Equal(t, "globstar\ton\n", buff.String())
}

func TestSplitArgs(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/split_args"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(true),
	)
	require.NoError(t, e.Setup())

	vars := ast.NewVars()
	vars.Set("CLI_ARGS", ast.Var{Value: "foo bar 'foo bar baz'"})

	err := e.Run(t.Context(), &task.Call{Task: "default", Vars: vars})
	require.NoError(t, err)
	assert.Equal(t, "3\n", buff.String())
}

func TestSingleCmdDep(t *testing.T) {
	t.Parallel()

	tt := fileContentTest{
		Dir:    "testdata/single_cmd_dep",
		Target: "foo",
		Files: map[string]string{
			"foo.txt": "foo\n",
			"bar.txt": "bar\n",
		},
	}
	t.Run("", func(t *testing.T) {
		t.Parallel()
		tt.Run(t)
	})
}

func TestSilence(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir("testdata/silent"),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithSilent(false),
	)
	require.NoError(t, e.Setup())

	// First verify that the silent flag is in place.
	fetchedTask, err := e.GetTask(&task.Call{Task: "task-test-silent-calls-chatty-silenced"})
	require.NoError(t, err, "Unable to look up task task-test-silent-calls-chatty-silenced")
	require.True(t, fetchedTask.Cmds[0].Silent, "The task task-test-silent-calls-chatty-silenced should have a silent call to chatty")

	// Then test the two basic cases where the task is silent or not.
	// A silenced task.
	err = e.Run(t.Context(), &task.Call{Task: "silent"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "siWhile running lent: Expected not see output, because the task is silent")

	buff.Reset()

	// A chatty (not silent) task.
	err = e.Run(t.Context(), &task.Call{Task: "chatty"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "chWhile running atty: Expected to see output, because the task is not silent")

	buff.Reset()

	// Then test invoking the two task from other tasks.
	// A silenced task that calls a chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-silent-calls-chatty-non-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-silent-calls-chatty-non-silenced: Expected to see output. The task is silenced, but the called task is not. Silence does not propagate to called tasks.")

	buff.Reset()

	// A silent task that does a silent call to a chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-silent-calls-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-silent-calls-chatty-silenced: Expected not to see output. The task calls chatty task, but the call is silenced.")

	buff.Reset()

	// A chatty task that does a call to a chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-chatty-calls-chatty-non-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-chatty-calls-chatty-non-silenced: Expected to see output. Both caller and callee are chatty and not silenced.")

	buff.Reset()

	// A chatty task that does a silenced call to a chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-chatty-calls-chatty-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-chatty-calls-chatty-silenced: Expected to see output. Call to a chatty task is silenced, but the parent task is not.")

	buff.Reset()

	// A chatty task with no cmd's of its own that does a silenced call to a chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-no-cmds-calls-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-no-cmds-calls-chatty-silenced: Expected not to see output. While the task itself is not silenced, it does not have any cmds and only does an invocation of a silenced task.")

	buff.Reset()

	// A chatty task that does a silenced invocation of a task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-chatty-calls-silenced-cmd"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-chatty-calls-silenced-cmd: Expected not to see output. While the task itself is not silenced, its call to the chatty task is silent.")

	buff.Reset()

	// Then test calls via dependencies.
	// A silent task that depends on a chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-is-silent-depends-on-chatty-non-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-is-silent-depends-on-chatty-non-silenced: Expected to see output. The task is silent and depends on a chatty task. Dependencies does not inherit silence.")

	buff.Reset()

	// A silent task that depends on a silenced chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-is-silent-depends-on-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-is-silent-depends-on-chatty-silenced: Expected not to see output. The task is silent and has a silenced dependency on a chatty task.")

	buff.Reset()

	// A chatty task that, depends on a silenced chatty task.
	err = e.Run(t.Context(), &task.Call{Task: "task-test-is-chatty-depends-on-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-is-chatty-depends-on-chatty-silenced: Expected not to see output. The task is chatty but does not have commands and has a silenced dependency on a chatty task.")

	buff.Reset()
}

func TestForce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		force    bool
		forceAll bool
	}{
		{
			name:  "force",
			force: true,
		},
		{
			name:     "force-all",
			forceAll: true,
		},
		{
			name:  "force with gentle force experiment",
			force: true,
			env: map[string]string{
				"TASK_X_GENTLE_FORCE": "1",
			},
		},
		{
			name:     "force-all with gentle force experiment",
			forceAll: true,
			env: map[string]string{
				"TASK_X_GENTLE_FORCE": "1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir("testdata/force"),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithForce(tt.force),
				task.WithForceAll(tt.forceAll),
			)
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(t.Context(), &task.Call{Task: "task-with-dep"}))
		})
	}
}

func TestWildcard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		call           string
		expectedOutput string
		wantErr        bool
	}{
		{
			name:           "basic wildcard",
			call:           "wildcard-foo",
			expectedOutput: "Hello foo\n",
		},
		{
			name:           "double wildcard",
			call:           "foo-wildcard-bar",
			expectedOutput: "Hello foo bar\n",
		},
		{
			name:           "store wildcard",
			call:           "start-foo",
			expectedOutput: "Starting foo\n",
		},
		{
			name:           "matches exactly",
			call:           "matches-exactly-*",
			expectedOutput: "I don't consume matches: []\n",
		},
		{
			name:    "no matches",
			call:    "no-match",
			wantErr: true,
		},
		{
			name:           "multiple matches",
			call:           "wildcard-foo-bar",
			expectedOutput: "Hello foo-bar\n",
		},
	}

	for _, test := range tests {
		t.Run(test.call, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			e := task.NewExecutor(
				task.WithDir("testdata/wildcards"),
				task.WithStdout(&buff),
				task.WithStderr(&buff),
				task.WithSilent(true),
				task.WithForce(true),
			)
			require.NoError(t, e.Setup())
			if test.wantErr {
				require.Error(t, e.Run(t.Context(), &task.Call{Task: test.call}))
				return
			}
			require.NoError(t, e.Run(t.Context(), &task.Call{Task: test.call}))
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

// enableExperimentForTest enables the experiment behind pointer e for the duration of test t and sub-tests,
// with the experiment being restored to its previous state when tests complete.
//
// Typically experiments are controlled via TASK_X_ env vars, but we cannot use those in tests
// because the experiment settings are parsed during experiments.init(), before any tests run.
func enableExperimentForTest(t *testing.T, e *experiments.Experiment, val int) {
	t.Helper()
	prev := *e
	*e = experiments.Experiment{
		Name:          prev.Name,
		AllowedValues: []int{val},
		Value:         val,
	}
	t.Cleanup(func() { *e = prev })
}
