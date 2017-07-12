package task

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mattn/go-zglob"
)

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
