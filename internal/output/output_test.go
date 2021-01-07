package output_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/output"
)

func TestInterleaved(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Interleaved{}
	var w = o.WrapWriter(&b, "")

	fmt.Fprintln(w, "foo\nbar")
	assert.Equal(t, "foo\nbar\n", b.String())
	fmt.Fprintln(w, "baz")
	assert.Equal(t, "foo\nbar\nbaz\n", b.String())
}

func TestGroup(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Group{}
	var w = o.WrapWriter(&b, "").(io.WriteCloser)

	fmt.Fprintln(w, "foo\nbar")
	assert.Equal(t, "", b.String())
	fmt.Fprintln(w, "baz")
	assert.Equal(t, "", b.String())
	assert.NoError(t, w.Close())
	assert.Equal(t, "foo\nbar\nbaz\n", b.String())
}

func TestPrefixed(t *testing.T) {
	var b bytes.Buffer
	var o output.Output = output.Prefixed{}
	var w = o.WrapWriter(&b, "prefix").(io.WriteCloser)

	t.Run("simple use cases", func(t *testing.T) {
		b.Reset()

		fmt.Fprintln(w, "foo\nbar")
		assert.Equal(t, "[prefix] foo\n[prefix] bar\n", b.String())
		fmt.Fprintln(w, "baz")
		assert.Equal(t, "[prefix] foo\n[prefix] bar\n[prefix] baz\n", b.String())
	})

	t.Run("multiple writes for a single line", func(t *testing.T) {
		b.Reset()

		for _, char := range []string{"T", "e", "s", "t", "!"} {
			fmt.Fprint(w, char)
			assert.Equal(t, "", b.String())
		}

		assert.NoError(t, w.Close())
		assert.Equal(t, "[prefix] Test!\n", b.String())
	})
}
