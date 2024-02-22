package fingerprint

import (
	"os"
	"path/filepath"
	"time"

	"github.com/go-task/task/v3/taskfile/ast"
)

// TimestampChecker checks if any source change compared with the generated files,
// using file modifications timestamps.
type TimestampChecker struct {
	tempDir string
	dry     bool
}

func NewTimestampChecker(tempDir string, dry bool) *TimestampChecker {
	return &TimestampChecker{
		tempDir: tempDir,
		dry:     dry,
	}
}

// IsUpToDate implements the Checker interface
func (checker *TimestampChecker) IsUpToDate(t *ast.Task) (bool, error) {
	if len(t.Sources) == 0 {
		return false, nil
	}

	sources, err := Globs(t.Dir, t.Sources)
	if err != nil {
		return false, nil
	}
	generates, err := Globs(t.Dir, t.Generates)
	if err != nil {
		return false, nil
	}

	timestampFile := checker.timestampFilePath(t)

	// If the file exists, add the file path to the generates.
	// If the generate file is old, the task will be executed.
	_, err = os.Stat(timestampFile)
	if err == nil {
		generates = append(generates, timestampFile)
	}
	// Compare the time of the generates and sources. If the generates are old, the task will be executed.

	// Get the max time of the generates.
	generateMaxTime, err := getMaxTime(generates...)
	if err != nil || generateMaxTime.IsZero() {
		return false, nil
	}

	// Check if any of the source files is newer than the max time of the generates.
	shouldUpdate, err := anyFileNewerThan(sources, generateMaxTime)
	if err != nil {
		return false, nil
	}

	return !shouldUpdate, nil
}

func (checker *TimestampChecker) Update(t *ast.Task) error {
	if !checker.dry {
		generates, err := Globs(t.Dir, t.Generates)
		if err != nil {
			return nil
		}
		if len(generates) == 0 {
			return nil
		}

		timestampFile := checker.timestampFilePath(t)
		_, err = os.Stat(timestampFile)
		if err == nil {
			// Modify the metadata of the file to the the current time.
			taskTime := time.Now()
			return os.Chtimes(timestampFile, taskTime, taskTime)
		}

		// Compare the time of the generates and sources. If the generates are old, the task will be executed.
		err = os.MkdirAll(filepath.Dir(timestampFile), 0o755)
		if err != nil {
			return err
		}
		f, err := os.Create(timestampFile)
		if err != nil {
			return err
		}
		return f.Close()
	}
	return nil
}

func (checker *TimestampChecker) Kind() string {
	return "timestamp"
}

// Value implements the Checker Interface
func (checker *TimestampChecker) Value(t *ast.Task) (any, error) {
	sources, err := Globs(t.Dir, t.Sources)
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

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// If the modification time of any of the files is newer than the the given time, returns true.
// This function is lazy, as it stops when it finds a file newer than the given time.
func anyFileNewerThan(files []string, givenTime time.Time) (bool, error) {
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return false, err
		}
		if info.ModTime().After(givenTime) {
			return true, nil
		}
	}
	return false, nil
}

// OnError implements the Checker interface
func (*TimestampChecker) OnError(t *ast.Task) error {
	return nil
}

func (checker *TimestampChecker) timestampFilePath(t *ast.Task) string {
	return filepath.Join(checker.tempDir, "timestamp", normalizeFilename(t.Task))
}
