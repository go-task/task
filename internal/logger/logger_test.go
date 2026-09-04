package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptIgnoresTerminalChatterInTheAnswer(t *testing.T) {
	t.Parallel()

	// A terminal answering a capability query in line mode puts its reply in
	// front of what the user typed.
	answer := "\x1b[?2026;4$y\x1b[?2027;0$yy\n"

	var out bytes.Buffer
	l := &Logger{
		Stdin:      strings.NewReader(answer),
		Stdout:     &out,
		Stderr:     &out,
		AssumeTerm: true,
	}

	require.NoError(t, l.Prompt(Default, "Really?", "n", "y", "yes"))
	assert.Contains(t, out.String(), "Really?")
}

func TestPromptRejectsAnAnswerThatIsNotAContinueValue(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	l := &Logger{
		Stdin:      strings.NewReader("no\n"),
		Stdout:     &out,
		Stderr:     &out,
		AssumeTerm: true,
	}

	assert.ErrorIs(t, l.Prompt(Default, "Really?", "n", "y", "yes"), ErrPromptCancelled)
}
