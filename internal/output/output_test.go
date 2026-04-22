package output_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestInterleaved(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	var o output.Output = output.Interleaved{}
	w, _, _ := o.WrapWriter(&b, io.Discard, "", nil)

	fmt.Fprintln(w, "foo\nbar")
	assert.Equal(t, "foo\nbar\n", b.String())
	fmt.Fprintln(w, "baz")
	assert.Equal(t, "foo\nbar\nbaz\n", b.String())
}

func TestGroup(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	var o output.Output = output.Group{}
	stdOut, stdErr, cleanup := o.WrapWriter(&b, io.Discard, "", nil)

	fmt.Fprintln(stdOut, "out\nout")
	assert.Equal(t, "", b.String())
	fmt.Fprintln(stdErr, "err\nerr")
	assert.Equal(t, "", b.String())
	fmt.Fprintln(stdOut, "out")
	assert.Equal(t, "", b.String())
	fmt.Fprintln(stdErr, "err")
	assert.Equal(t, "", b.String())

	require.NoError(t, cleanup(nil))
	assert.Equal(t, "out\nout\nerr\nerr\nout\nerr\n", b.String())
}

func TestGroupWithBeginEnd(t *testing.T) {
	t.Parallel()

	tmpl := templater.Cache{
		Vars: ast.NewVars(
			&ast.VarElement{
				Key:   "VAR1",
				Value: ast.Var{Value: "example-value"},
			},
		),
	}

	var o output.Output = output.Group{
		Begin: "::group::{{ .VAR1 }}",
		End:   "::endgroup::",
	}
	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		var b bytes.Buffer
		w, _, cleanup := o.WrapWriter(&b, io.Discard, "", &tmpl)

		fmt.Fprintln(w, "foo\nbar")
		assert.Equal(t, "", b.String())
		fmt.Fprintln(w, "baz")
		assert.Equal(t, "", b.String())
		require.NoError(t, cleanup(nil))
		assert.Equal(t, "::group::example-value\nfoo\nbar\nbaz\n::endgroup::\n", b.String())
	})
	t.Run("no output", func(t *testing.T) {
		t.Parallel()

		var b bytes.Buffer
		_, _, cleanup := o.WrapWriter(&b, io.Discard, "", &tmpl)
		require.NoError(t, cleanup(nil))
		assert.Equal(t, "", b.String())
	})
}

func TestGroupErrorOnlySwallowsOutputOnNoError(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	var o output.Output = output.Group{
		ErrorOnly: true,
	}
	stdOut, stdErr, cleanup := o.WrapWriter(&b, io.Discard, "", nil)

	_, _ = fmt.Fprintln(stdOut, "std-out")
	_, _ = fmt.Fprintln(stdErr, "std-err")

	require.NoError(t, cleanup(nil))
	assert.Empty(t, b.String())
}

func TestGroupErrorOnlyShowsOutputOnError(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	var o output.Output = output.Group{
		ErrorOnly: true,
	}
	stdOut, stdErr, cleanup := o.WrapWriter(&b, io.Discard, "", nil)

	_, _ = fmt.Fprintln(stdOut, "std-out")
	_, _ = fmt.Fprintln(stdErr, "std-err")

	require.NoError(t, cleanup(errors.New("any-error")))
	assert.Equal(t, "std-out\nstd-err\n", b.String())
}

func gitlabTaskCache(taskName string) *templater.Cache {
	return &templater.Cache{
		Vars: ast.NewVars(
			&ast.VarElement{
				Key:   "TASK",
				Value: ast.Var{Value: taskName},
			},
		),
	}
}

var gitlabMarkerPattern = regexp.MustCompile(
	`\x1b\[0Ksection_start:(\d+):(\S+?)(\[[^\]]+\])?\r\x1b\[0K(.*)\n` +
		`(?s)(.*)` +
		`\x1b\[0Ksection_end:(\d+):(\S+)\r\x1b\[0K\n$`,
)

func TestGitLab(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{}
	w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("build"))

	fmt.Fprintln(w, "hello")
	assert.Equal(t, "", b.String(), "output must be buffered until close")
	require.NoError(t, cleanup(nil))

	m := gitlabMarkerPattern.FindStringSubmatch(b.String())
	require.NotNil(t, m, "output should match GitLab section markers, got: %q", b.String())
	assert.Equal(t, m[2], m[7], "start and end section IDs must match")
	assert.Empty(t, m[3], "collapsed option should not be present by default")
	assert.Equal(t, "build", m[4], "section header should be the task name")
	assert.Equal(t, "hello\n", m[5], "wrapped content must be preserved")
	assert.Contains(t, m[2], "build_", "section ID should be prefixed with slugged task name")
}

func TestGitLabUniqueSectionIDs(t *testing.T) {
	t.Parallel()

	o := output.GitLab{}

	ids := make([]string, 3)
	for i := range ids {
		var b bytes.Buffer
		w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("build"))
		fmt.Fprintln(w, "x")
		require.NoError(t, cleanup(nil))
		m := gitlabMarkerPattern.FindStringSubmatch(b.String())
		require.NotNil(t, m)
		ids[i] = m[2]
	}

	assert.NotEqual(t, ids[0], ids[1])
	assert.NotEqual(t, ids[1], ids[2])
	assert.NotEqual(t, ids[0], ids[2])
}

func TestGitLabCollapsed(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{Collapsed: true}
	w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("build"))
	fmt.Fprintln(w, "x")
	require.NoError(t, cleanup(nil))

	m := gitlabMarkerPattern.FindStringSubmatch(b.String())
	require.NotNil(t, m)
	assert.Equal(t, "[collapsed=true]", m[3])
}

func TestGitLabErrorOnlySwallowsOutputOnNoError(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{ErrorOnly: true}
	w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("build"))
	fmt.Fprintln(w, "hello")
	require.NoError(t, cleanup(nil))
	assert.Empty(t, b.String())
}

func TestGitLabErrorOnlyShowsOutputOnError(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{ErrorOnly: true}
	w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("build"))
	fmt.Fprintln(w, "hello")
	require.NoError(t, cleanup(errors.New("boom")))

	m := gitlabMarkerPattern.FindStringSubmatch(b.String())
	require.NotNil(t, m)
	assert.Equal(t, "hello\n", m[5])
}

func TestGitLabSlugSanitizesTaskName(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{}
	w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("my task:with spaces"))
	fmt.Fprintln(w, "x")
	require.NoError(t, cleanup(nil))

	m := gitlabMarkerPattern.FindStringSubmatch(b.String())
	require.NotNil(t, m)
	assert.Regexp(t, `^[a-zA-Z0-9_.-]+$`, m[2], "section ID must only contain GitLab-allowed chars")
}

func TestGitLabWrapWriterIsPassthrough(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{}
	w, _, cleanup := o.WrapWriter(&b, io.Discard, "", nil)

	fmt.Fprintln(w, "hello")
	assert.Equal(t, "hello\n", b.String(), "WrapWriter must be a passthrough for GitLab")
	assert.NoError(t, cleanup(nil))
	assert.Equal(t, "hello\n", b.String(), "closer must be a no-op")
}

func TestGitLabWrapTaskSingleSection(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{}
	w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("build"))

	// Simulate multiple cmd outputs being written during a task's execution.
	fmt.Fprintln(w, "cmd 1 output")
	fmt.Fprintln(w, "cmd 2 output")
	fmt.Fprintln(w, "cmd 3 output")
	require.NoError(t, cleanup(nil))

	// There must be exactly one section_start and one section_end.
	assert.Equal(t, 1, strings.Count(b.String(), "section_start:"))
	assert.Equal(t, 1, strings.Count(b.String(), "section_end:"))

	m := gitlabMarkerPattern.FindStringSubmatch(b.String())
	require.NotNil(t, m)
	assert.Equal(t, "cmd 1 output\ncmd 2 output\ncmd 3 output\n", m[5])
}

func TestGitLabWrapTaskDurationElapsed(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	o := output.GitLab{}
	w, _, cleanup := o.WrapTask(&b, io.Discard, gitlabTaskCache("slow"))

	fmt.Fprintln(w, "started")
	time.Sleep(1100 * time.Millisecond)
	fmt.Fprintln(w, "done")
	require.NoError(t, cleanup(nil))

	m := gitlabMarkerPattern.FindStringSubmatch(b.String())
	require.NotNil(t, m)
	startTS, err := strconv.ParseInt(m[1], 10, 64)
	require.NoError(t, err)
	endTS, err := strconv.ParseInt(m[6], 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, endTS-startTS, int64(1),
		"end TS must be at least 1 second after start TS when task takes >1s")
}

func TestGitLabWrapTaskNested(t *testing.T) {
	t.Parallel()

	var root bytes.Buffer
	parent := output.GitLab{}
	parentW, _, parentClose := parent.WrapTask(&root, io.Discard, gitlabTaskCache("parent"))

	fmt.Fprintln(parentW, "before child")

	child := output.GitLab{}
	childW, _, childClose := child.WrapTask(parentW, io.Discard, gitlabTaskCache("child"))
	fmt.Fprintln(childW, "inside child")
	require.NoError(t, childClose(nil))

	fmt.Fprintln(parentW, "after child")
	require.NoError(t, parentClose(nil))

	out := root.String()
	// Two section_start and two section_end
	assert.Equal(t, 2, strings.Count(out, "section_start:"))
	assert.Equal(t, 2, strings.Count(out, "section_end:"))

	// Order: parent start → child start → child end → parent end
	parentStart := strings.Index(out, "section_start:") // first
	childStart := strings.Index(out[parentStart+1:], "section_start:") + parentStart + 1
	childEnd := strings.Index(out, "section_end:")
	parentEnd := strings.LastIndex(out, "section_end:")
	assert.Less(t, parentStart, childStart, "child_start must come after parent_start")
	assert.Less(t, childStart, childEnd, "child_end must come after child_start")
	assert.Less(t, childEnd, parentEnd, "parent_end must come after child_end")
}

func TestGitLabWrapTaskConcurrentWrites(t *testing.T) {
	t.Parallel()

	var root bytes.Buffer
	parent := output.GitLab{}
	parentW, _, parentClose := parent.WrapTask(&root, io.Discard, gitlabTaskCache("parent"))

	const numChildren = 10
	var wg sync.WaitGroup
	for i := 0; i < numChildren; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			child := output.GitLab{}
			childW, _, childClose := child.WrapTask(parentW, io.Discard, gitlabTaskCache(fmt.Sprintf("child%d", i)))
			fmt.Fprintf(childW, "child %d output\n", i)
			_ = childClose(nil)
		}(i)
	}
	wg.Wait()
	require.NoError(t, parentClose(nil))

	out := root.String()
	// 1 parent + 10 children = 11 section_start and 11 section_end
	assert.Equal(t, 11, strings.Count(out, "section_start:"))
	assert.Equal(t, 11, strings.Count(out, "section_end:"))
	// All 10 child outputs present
	for i := 0; i < numChildren; i++ {
		assert.Contains(t, out, fmt.Sprintf("child %d output", i))
	}
}

func TestPrefixed(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	var b bytes.Buffer
	l := &logger.Logger{
		Color: false,
	}

	var o output.Output = output.NewPrefixed(l)
	w, _, cleanup := o.WrapWriter(&b, io.Discard, "prefix", nil)

	t.Run("simple use cases", func(t *testing.T) { //nolint:paralleltest // cannot run in parallel
		b.Reset()

		fmt.Fprintln(w, "foo\nbar")
		assert.Equal(t, "[prefix] foo\n[prefix] bar\n", b.String())
		fmt.Fprintln(w, "baz")
		assert.Equal(t, "[prefix] foo\n[prefix] bar\n[prefix] baz\n", b.String())
		require.NoError(t, cleanup(nil))
	})

	t.Run("multiple writes for a single line", func(t *testing.T) { //nolint:paralleltest // cannot run in parallel
		b.Reset()

		for _, char := range []string{"T", "e", "s", "t", "!"} {
			fmt.Fprint(w, char)
			assert.Equal(t, "", b.String())
		}

		require.NoError(t, cleanup(nil))
		assert.Equal(t, "[prefix] Test!\n", b.String())
	})
}

func TestPrefixedWithColor(t *testing.T) {
	t.Parallel()

	color.NoColor = false

	var b bytes.Buffer
	l := &logger.Logger{
		Color: true,
	}

	var o output.Output = output.NewPrefixed(l)

	writers := make([]io.Writer, 16)
	for i := range writers {
		writers[i], _, _ = o.WrapWriter(&b, io.Discard, fmt.Sprintf("prefix-%d", i), nil)
	}

	t.Run("colors should loop", func(t *testing.T) {
		t.Parallel()

		for i, w := range writers {
			b.Reset()

			color := output.PrefixColorSequence[i%len(output.PrefixColorSequence)]

			var prefix bytes.Buffer
			l.FOutf(&prefix, color, fmt.Sprintf("prefix-%d", i))

			fmt.Fprintln(w, "foo\nbar")
			assert.Equal(
				t,
				fmt.Sprintf("[%s] foo\n[%s] bar\n", prefix.String(), prefix.String()),
				b.String(),
			)
		}
	})
}
