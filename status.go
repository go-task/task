package task

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/go-task/task/execext"
	"github.com/mattn/go-zglob"
)

func (e *Executor) isTaskUpToDate(ctx context.Context, t *Task) (bool, error) {
	if len(t.Status) > 0 {
		return e.isUpToDateStatus(ctx, t)
	}
	return e.isUpToDateTimestamp(ctx, t)
}

func (e *Executor) isUpToDateStatus(ctx context.Context, t *Task) (bool, error) {
	for _, s := range t.Status {
		err := execext.RunCommand(&execext.RunCommandOptions{
			Context: ctx,
			Command: s,
			Dir:     e.getTaskDir(t),
			Env:     e.getEnviron(t),
		})
		if err != nil {
			return false, nil
		}
	}
	return true, nil
}

func (e *Executor) isUpToDateTimestamp(ctx context.Context, t *Task) (bool, error) {
	if len(t.Sources) == 0 || len(t.Generates) == 0 {
		return false, nil
	}

	dir := e.getTaskDir(t)

	sourcesMaxTime, err := getPatternsMaxTime(dir, t.Sources)
	if err != nil || sourcesMaxTime.IsZero() {
		return false, nil
	}

	generatesMinTime, err := getPatternsMinTime(dir, t.Generates)
	if err != nil || generatesMinTime.IsZero() {
		return false, nil
	}
	return !generatesMinTime.Before(sourcesMaxTime), nil
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

func getPatternsMinTime(dir string, patterns []string) (m time.Time, err error) {
	for _, p := range patterns {
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		mp, err := getPatternMinTime(p)
		if err != nil {
			return time.Time{}, err
		}
		m = minTime(m, mp)
	}
	return
}
func getPatternsMaxTime(dir string, patterns []string) (m time.Time, err error) {
	for _, p := range patterns {
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		mp, err := getPatternMaxTime(p)
		if err != nil {
			return time.Time{}, err
		}
		m = maxTime(m, mp)
	}
	return
}

func getPatternMinTime(pattern string) (minTime time.Time, err error) {
	files, err := zglob.Glob(pattern)
	if err != nil {
		return time.Time{}, err
	}

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return time.Time{}, err
		}

		modTime := info.ModTime()
		if minTime.IsZero() || modTime.Before(minTime) {
			minTime = modTime
		}
	}
	return
}

func getPatternMaxTime(pattern string) (maxTime time.Time, err error) {
	files, err := zglob.Glob(pattern)
	if err != nil {
		return time.Time{}, err
	}

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return time.Time{}, err
		}

		modTime := info.ModTime()
		if modTime.After(maxTime) {
			maxTime = modTime
		}
	}
	return
}
