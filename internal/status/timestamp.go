package status

import (
	"os"
	"time"
)

// Timestamp checks if any source change compared with the generated files,
// using file modifications timestamps.
type Timestamp struct {
	Dir       string
	Sources   []string
	Generates []string
}

// IsUpToDate implements the Checker interface
func (t *Timestamp) IsUpToDate() (bool, error) {
	if len(t.Sources) == 0 || len(t.Generates) == 0 {
		return false, nil
	}

	sources, err := globs(t.Dir, t.Sources)
	if err != nil {
		return false, nil
	}
	generates, err := globs(t.Dir, t.Generates)
	if err != nil {
		return false, nil
	}

	sourcesMaxTime, err := getMaxTime(sources...)
	if err != nil || sourcesMaxTime.IsZero() {
		return false, nil
	}

	generatesMinTime, err := getMinTime(generates...)
	if err != nil || generatesMinTime.IsZero() {
		return false, nil
	}

	return !generatesMinTime.Before(sourcesMaxTime), nil
}

func (t *Timestamp) Kind() string {
	return "timestamp"
}

// Value implements the Checker Interface
func (t *Timestamp) Value() (interface{}, error) {
	sources, err := globs(t.Dir, t.Sources)
	if err != nil {
		return time.Now(), err
	}

	sourcesMaxTime, err := getMaxTime(sources...)
	if err != nil {
		return time.Now(), err
	}

	if sourcesMaxTime.IsZero() {
		return time.Unix(0, 0), nil
	}

	return sourcesMaxTime, nil
}

func getMinTime(files ...string) (time.Time, error) {
	var t time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return time.Time{}, err
		}
		t = minTime(t, info.ModTime())
	}
	return t, nil
}

func getMaxTime(files ...string) (time.Time, error) {
	var t time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return time.Time{}, err
		}
		t = maxTime(t, info.ModTime())
	}
	return t, nil
}

func minTime(a, b time.Time) time.Time {
	if !a.IsZero() && a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// OnError implements the Checker interface
func (*Timestamp) OnError() error {
	return nil
}
