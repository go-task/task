package output_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile"
)

func TestInterleaved(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Interleaved{}
	var w, _, _ = o.WrapWriter(&b, io.Discard, "", nil)

	fmt.Fprintln(w, "foo\nbar")
	assert.Equal(t, "foo\nbar\n", b.String())
	fmt.Fprintln(w, "baz")
	assert.Equal(t, "foo\nbar\nbaz\n", b.String())
}

func TestGroup(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Group{}
	var stdOut, stdErr, cleanup = o.WrapWriter(&b, io.Discard, "", nil)

	fmt.Fprintln(stdOut, "out\nout")
	assert.Equal(t, "", b.String())
	fmt.Fprintln(stdErr, "err\nerr")
	assert.Equal(t, "", b.String())
	fmt.Fprintln(stdOut, "out")
	assert.Equal(t, "", b.String())
	fmt.Fprintln(stdErr, "err")
	assert.Equal(t, "", b.String())

	assert.NoError(t, cleanup(nil))
	assert.Equal(t, "out\nout\nerr\nerr\nout\nerr\n", b.String())
}

func TestGroupWithBeginEnd(t *testing.T) {
	tmpl := templater.Templater{
		Vars: &taskfile.Vars{
			Keys: []string{"VAR1"},
			Mapping: map[string]taskfile.Var{
				"VAR1": {Static: "example-value"},
			},
		},
	}

	var o output.Output = output.Group{
		Begin: "::group::{{ .VAR1 }}",
		End:   "::endgroup::",
	}
	t.Run("simple", func(t *testing.T) {
		var b bytes.Buffer
		var w, _, cleanup = o.WrapWriter(&b, io.Discard, "", &tmpl)

		fmt.Fprintln(w, "foo\nbar")
		assert.Equal(t, "", b.String())
		fmt.Fprintln(w, "baz")
		assert.Equal(t, "", b.String())
		assert.NoError(t, cleanup(nil))
		assert.Equal(t, "::group::example-value\nfoo\nbar\nbaz\n::endgroup::\n", b.String())
	})
	t.Run("no output", func(t *testing.T) {
		var b bytes.Buffer
		var _, _, cleanup = o.WrapWriter(&b, io.Discard, "", &tmpl)
		assert.NoError(t, cleanup(nil))
		assert.Equal(t, "", b.String())
	})
}

func TestGroupErrorOnlySwallowsOutputOnNoError(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Group{
		ErrorOnly: true,
	}
	var stdOut, stdErr, cleanup = o.WrapWriter(&b, io.Discard, "", nil)

	_, _ = fmt.Fprintln(stdOut, "std-out")
	_, _ = fmt.Fprintln(stdErr, "std-err")

	assert.NoError(t, cleanup(nil))
	assert.Empty(t, b.String())
}
func TestGroupErrorOnlyShowsOutputOnError(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Group{
		ErrorOnly: true,
	}
	var stdOut, stdErr, cleanup = o.WrapWriter(&b, io.Discard, "", nil)

	_, _ = fmt.Fprintln(stdOut, "std-out")
	_, _ = fmt.Fprintln(stdErr, "std-err")

	assert.NoError(t, cleanup(errors.New("any-error")))
	assert.Equal(t, "std-out\nstd-err\n", b.String())
}

func TestPrefixed(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Prefixed{}
	var w, _, cleanup = o.WrapWriter(&b, io.Discard, "prefix", nil)

	t.Run("simple use cases", func(t *testing.T) {
		b.Reset()

		fmt.Fprintln(w, "foo\nbar")
		assert.Equal(t, "[prefix] foo\n[prefix] bar\n", b.String())
		fmt.Fprintln(w, "baz")
		assert.Equal(t, "[prefix] foo\n[prefix] bar\n[prefix] baz\n", b.String())
		assert.NoError(t, cleanup(nil))
	})

	t.Run("multiple writes for a single line", func(t *testing.T) {
		b.Reset()

		for _, char := range []string{"T", "e", "s", "t", "!"} {
			fmt.Fprint(w, char)
			assert.Equal(t, "", b.String())
		}

		assert.NoError(t, cleanup(nil))
		assert.Equal(t, "[prefix] Test!\n", b.String())
	})
}
