package output

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
)

// Status prints one line for each task. It hides the output of a task that is
// successful.
type Status struct {
	group  Group
	logger *logger.Logger
	mutex  sync.Mutex

	failed    int
	skipped   int
	succeeded int
}

func NewStatus(l *logger.Logger) *Status {
	return &Status{
		group:  Group{ErrorOnly: true},
		logger: l,
	}
}

func (s *Status) WrapWriter(stdOut, stdErr io.Writer, prefix string, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	return s.group.WrapWriter(stdOut, stdErr, prefix, cache)
}

func (s *Status) TaskStarted(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.printf(logger.Cyan, "Running", name, "")
}

func (s *Status) TaskSkipped(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.skipped++
	s.printf(logger.Yellow, "Skipped", name, "")
}

func (s *Status) TaskFinished(name string, err error, d time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err != nil {
		s.failed++
		s.printf(logger.Red, "Failed", name, formatDuration(d))
		return
	}

	s.succeeded++
	s.printf(logger.Green, "Succeeded", name, formatDuration(d))
}

func (s *Status) RunFinished() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var parts []string
	for _, c := range []struct {
		label string
		count int
	}{
		{"Skipped", s.skipped},
		{"Succeeded", s.succeeded},
		{"Failed", s.failed},
	} {
		if c.count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c.count, c.label))
		}
	}

	if len(parts) == 0 {
		return
	}

	s.logger.Errf(logger.Default, "%s\n", strings.Join(parts, ", "))
}

func (s *Status) printf(color logger.Color, state, name, duration string) {
	if duration != "" {
		duration = " (" + duration + ")"
	}
	s.logger.Errf(color, "%-10s %s%s\n", state, sanitize(name), duration)
}

// sanitize removes the control characters from a label. The `prefix` is free
// text from a Taskfile, which is frequently code from a different source. A
// control character moves the cursor and writes over a line above, thus a task
// with a failure can show itself as successful.
func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
