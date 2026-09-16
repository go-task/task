package task_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"maps"
	rand "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/filepathext"
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

// The injected fingerprint variable follows the method the up-to-date check
// uses, including when that method comes from the Taskfile level.
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

func TestIncludesHttp(t *testing.T) {
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
