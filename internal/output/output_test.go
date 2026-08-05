package output_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

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

// TestPrefixedConcurrentStdoutStderr is a regression test for
// https://github.com/go-task/task/issues/2945: WrapWriter returns the same
// *prefixWriter for both stdout and stderr, and os/exec copies command
// output to each of them from its own goroutine, so Write can legitimately
// be called concurrently from two goroutines for a single task. Since
// bytes.Buffer isn't safe for concurrent use, unsynchronized access to the
// shared buffer could corrupt its internal state and panic (observed as
// "slice bounds out of range"). Run with -race for a reliable signal; this
// also reproduces the panic reliably on an unfixed prefixWriter even
// without -race, given enough concurrent lines.
func TestPrefixedConcurrentStdoutStderr(t *testing.T) {
	t.Parallel()

	l := &logger.Logger{Color: false}
	var o output.Output = output.NewPrefixed(l)
	stdOut, stdErr, cleanup := o.WrapWriter(io.Discard, io.Discard, "prefix", nil)

	const lines = 5000
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range lines {
			fmt.Fprintf(stdOut, "stdout line %d\n", i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := range lines {
			fmt.Fprintf(stdErr, "stderr line %d\n", i)
		}
	}()
	wg.Wait()

	require.NoError(t, cleanup(nil))
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
