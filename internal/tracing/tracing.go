package tracing

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Tracer struct {
	mu      sync.Mutex
	spans   []*Span
	outFile string

	timeFn func() time.Time
}

func NewTracer(outFile string) Tracer {
	return Tracer{
		outFile: outFile,
	}
}

type Span struct {
	parent    *Tracer
	name      string
	startedAt time.Time
	endedAt   time.Time
}

func (t *Tracer) Start(name string) *Span {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timeFn == nil {
		t.timeFn = time.Now
	}

	result := &Span{
		parent:    t,
		name:      name,
		startedAt: t.timeFn(),
	}
	t.spans = append(t.spans, result)
	return result
}

func (s *Span) Stop() {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	s.endedAt = s.parent.timeFn()
}

func (t *Tracer) WriteOutput() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.outFile == "" {
		return nil
	}
	return os.WriteFile(t.outFile, []byte(t.toMermaidOutput()), 0644)
}

func (t *Tracer) toMermaidOutput() string {
	out := `gantt
    title Task Execution Timeline
    dateFormat YYYY-MM-DD HH:mm:ss.SSS
	axisFormat %X
`
	dateFormat := "2006-01-02 15:04:05.000"
	for _, span := range t.spans {
		if span.endedAt.IsZero() {
			continue
		}
		name := strings.Replace(span.name, ":", "|", -1)
		duration := span.endedAt.Sub(span.startedAt).Truncate(time.Millisecond * 100)
		out += fmt.Sprintf("    %s [%v] :done, %s, %s\n", name, duration, span.startedAt.Format(dateFormat), span.endedAt.Format(dateFormat))
	}

	return out
}
