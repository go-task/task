package tracing

import (
	"fmt"
	"strings"
	"time"
)

type Tracer struct {
	spans []*Span
}

type Span struct {
	name      string
	startedAt time.Time
	endedAt   time.Time
}

func (t *Tracer) Start(name string) *Span {
	result := &Span{
		name:      name,
		startedAt: time.Now(),
	}
	t.spans = append(t.spans, result)
	return result
}

func (s *Span) Stop() {
	s.endedAt = time.Now()
}

func (t *Tracer) ToMermaidOutput() string {
	out := `gantt
    title Task Execution Timeline
    dateFormat YYYY-MM-DD HH:mm:ss.SSS
	axisFormat %X
    section Tasks
`
	dateFormat := "2006-01-02 15:04:05.000"
	for _, span := range t.spans {
		if span.endedAt.IsZero() {
			continue
		}
		name := strings.Replace(span.name, ":", "|", -1)
		duration := span.endedAt.Sub(span.startedAt)
		out += fmt.Sprintf("    %s-%v :done, %s, %s\n", name, duration, span.startedAt.Format(dateFormat), span.endedAt.Format(dateFormat))
	}

	return out
}
