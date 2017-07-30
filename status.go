package task

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/go-task/task/execext"
	"github.com/mattn/go-zglob"
)

func (t *Task) isUpToDate(ctx context.Context) (bool, error) {
	if len(t.Status) > 0 {
		return t.isUpToDateStatus(ctx)
	}
	return t.isUpToDateTimestamp(ctx)
}

func (t *Task) isUpToDateStatus(ctx context.Context) (bool, error) {
	for _, s := range t.Status {
		err := execext.RunCommand(&execext.RunCommandOptions{
			Context: ctx,
			Command: s,
			Dir:     t.Dir,
			Env:     t.getEnviron(),
		})
		if err != nil {
			return false, nil
		}
	}
	return true, nil
}

func (t *Task) isUpToDateTimestamp(ctx context.Context) (bool, error) {
	if len(t.Sources) == 0 || len(t.Generates) == 0 {
		return false, nil
	}

	sourcesMaxTime, err := getPatternsMaxTime(t.Dir, t.Sources)
	if err != nil || sourcesMaxTime.IsZero() {
		return false, nil
	}

	generatesMinTime, err := getPatternsMinTime(t.Dir, t.Generates)
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
