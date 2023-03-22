package fingerprint

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile"
)

// ChecksumChecker validates if a task is up to date by calculating its source
// files checksum
type ChecksumChecker struct {
	tempDir string
	dry     bool
}

func NewChecksumChecker(tempDir string, dry bool) *ChecksumChecker {
	return &ChecksumChecker{
		tempDir: tempDir,
		dry:     dry,
	}
}

func (checker *ChecksumChecker) IsUpToDate(t *taskfile.Task) (bool, error) {
	if len(t.Sources) == 0 {
		return false, nil
	}

	checksumFile := checker.checksumFilePath(t)

	data, _ := os.ReadFile(checksumFile)
	oldMd5 := strings.TrimSpace(string(data))

	newMd5, err := checker.checksum(t)
	if err != nil {
		return false, nil
	}

	if !checker.dry {
		_ = os.MkdirAll(filepathext.SmartJoin(checker.tempDir, "checksum"), 0o755)
		if err = os.WriteFile(checksumFile, []byte(newMd5+"\n"), 0o644); err != nil {
			return false, err
		}
	}

	if len(t.Generates) > 0 {
		// For each specified 'generates' field, check whether the files actually exist
		for _, g := range t.Generates {
			generates, err := Glob(t.Dir, g)
			if os.IsNotExist(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			if len(generates) == 0 {
				return false, nil
			}
		}
	}

	return oldMd5 == newMd5, nil
}

func (checker *ChecksumChecker) Value(t *taskfile.Task) (interface{}, error) {
	return checker.checksum(t)
}

func (checker *ChecksumChecker) OnError(t *taskfile.Task) error {
	if len(t.Sources) == 0 {
		return nil
	}
	return os.Remove(checker.checksumFilePath(t))
}

func (*ChecksumChecker) Kind() string {
	return "checksum"
}

func (c *ChecksumChecker) checksum(t *taskfile.Task) (string, error) {
	sources, err := globs(t.Dir, t.Sources)
	if err != nil {
		return "", err
	}

	h := md5.New()
	for _, f := range sources {
		// also sum the filename, so checksum changes for renaming a file
		if _, err := io.Copy(h, strings.NewReader(filepath.Base(f))); err != nil {
			return "", err
		}
		f, err := os.Open(f)
		if err != nil {
			return "", err
		}
		if _, err = io.Copy(h, f); err != nil {
			return "", err
		}
		f.Close()
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (checker *ChecksumChecker) checksumFilePath(t *taskfile.Task) string {
	return filepath.Join(checker.tempDir, "checksum", normalizeFilename(t.Name()))
}

var checksumFilenameRegexp = regexp.MustCompile("[^A-z0-9]")

// replaces invalid characters on filenames with "-"
func normalizeFilename(f string) string {
	return checksumFilenameRegexp.ReplaceAllString(f, "-")
}
