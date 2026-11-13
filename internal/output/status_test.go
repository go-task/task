package output_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
)

func TestStatusReportsATaskThatSucceeds(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: &b, Color: false})

	s.TaskStarted("build")
	s.TaskFinished("build", nil, 1500*time.Millisecond)

	assert.Equal(t, "Running    build\nSucceeded  build (1.50s)\n", b.String())
}

func TestStatusDoesNotCountASkippedTaskAsSucceeded(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: &b, Color: false})

	s.TaskStarted("build")
	s.TaskSkipped("build")
	s.RunFinished()

	assert.Equal(t, "Running    build\nSkipped    build\n1 Skipped\n", b.String())
}

// A `for` loop can start one task two or more times with the same label.
func TestStatusReportsEveryTaskThatSharesALabel(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: &b, Color: false})

	s.TaskStarted("credential")
	s.TaskStarted("credential")
	s.TaskFinished("credential", nil, time.Second)
	s.TaskFinished("credential", errors.New("boom"), time.Second)
	s.RunFinished()

	assert.Contains(t, b.String(), "Failed     credential")
	assert.Contains(t, b.String(), "1 Succeeded, 1 Failed")
}

func TestStatusRemovesControlCharactersFromALabel(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: &b, Color: false})

	s.TaskStarted("real\x1b[2A\x1b[2KSucceeded  fake")
	s.TaskSkipped("two\nlines")

	assert.Equal(t, "Running    real[2A[2KSucceeded  fake\nSkipped    twolines\n", b.String())
}

func TestStatusDoesNotReadALabelAsAFormatString(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: &b, Color: false})

	s.TaskStarted("%s%d%!(EXTRA")

	assert.Equal(t, "Running    %s%d%!(EXTRA\n", b.String())
}

func TestStatusReportsATaskThatFails(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: &b, Color: false})

	s.TaskStarted("build")
	s.TaskFinished("build", errors.New("boom"), 250*time.Microsecond)
	s.RunFinished()

	assert.Equal(t, "Running    build\nFailed     build (0.25ms)\n1 Failed\n", b.String())
}

func TestStatusPrintsNoSummaryWhenNoTaskRan(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: &b, Color: false})

	s.RunFinished()

	assert.Empty(t, b.String())
}

func TestStatusHidesTheOutputOfATaskThatSucceeds(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: io.Discard, Color: false})
	stdOut, stdErr, cleanup := s.WrapWriter(&out, io.Discard, "build", nil)

	fmt.Fprintln(stdOut, "compiling")
	fmt.Fprintln(stdErr, "warning")

	require.NoError(t, cleanup(nil))
	assert.Empty(t, out.String())
}

func TestStatusShowsTheWholeOutputOfATaskThatFails(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	s := output.NewStatus(&logger.Logger{Stderr: io.Discard, Color: false})
	stdOut, stdErr, cleanup := s.WrapWriter(&out, io.Discard, "build", nil)

	fmt.Fprintln(stdOut, "compiling")
	fmt.Fprintln(stdErr, "warning")

	require.NoError(t, cleanup(errors.New("boom")))
	assert.Equal(t, "compiling\nwarning\n", out.String())
}
