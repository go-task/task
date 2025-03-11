package tracing

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestTracer_Start(t *testing.T) {
	tracer := NewTracer(t.TempDir() + "/tracing.txt")

	currentTime, err := time.Parse(time.DateTime, "2025-01-02 15:42:23")
	require.NoError(t, err)
	tracer.timeFn = func() time.Time {
		return currentTime
	}

	task1 := tracer.Start("task one")
	currentTime = currentTime.Add(time.Second)

	// special chars handling: will be replaced with "namespace|task two" in the output
	task2 := tracer.Start("namespace:task two")
	tracer.Start("task three - did not finish, should not show up in end result")
	currentTime = currentTime.Add(time.Second * 2)

	task1.Stop()
	currentTime = currentTime.Add(time.Second * 3)
	task2.Stop()

	// very short tasks should still show up as a point in timeline
	tracer.Start("very short task").Stop()

	r := require.New(t)
	r.NoError(tracer.WriteOutput())

	contents, err := os.ReadFile(tracer.outFile)
	r.NoError(err)

	expectedContents := `gantt
    title Task Execution Timeline
    dateFormat YYYY-MM-DD HH:mm:ss.SSS
	axisFormat %X
    task one [3s] :done, 2025-01-02 15:42:23.000, 2025-01-02 15:42:26.000
    namespace|task two [5s] :done, 2025-01-02 15:42:24.000, 2025-01-02 15:42:29.000
    very short task [0s] :done, 2025-01-02 15:42:29.000, 2025-01-02 15:42:29.000
`

	r.Equal(expectedContents, string(contents))
}
