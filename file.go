package task

import (
	"os"
	"time"

	"github.com/mattn/go-zglob"
)

var dirsToSkip = []string{
	".git",
	"node_modules",
}

func minTime(pattern string) (minTime time.Time, err error) {
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

func maxTime(pattern string) (maxTime time.Time, err error) {
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
