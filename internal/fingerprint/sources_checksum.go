package fingerprint

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zeebo/xxh3"

	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile/ast"
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

func (checker *ChecksumChecker) IsUpToDate(t *ast.Task) (bool, string, error) {
	if len(t.Sources) == 0 && len(t.Generates) == 0 {
		return false, "", nil
	}

	checksumFile := checker.checksumFilePath(t)

	data, _ := os.ReadFile(checksumFile)
	oldHashes := strings.TrimSpace(string(data))
	oldSourcesHash, oldGeneratesdHash, _ := strings.Cut(oldHashes, "\n")

	newSourcesHash, err := checker.checksum(t, t.Sources)
	if err != nil {
		return false, "", err
	}

	newGeneratesHash, err := checker.checksum(t, t.Generates)
	if err != nil {
		return false, "", err
	}

	return oldSourcesHash == newSourcesHash && oldGeneratesdHash == newGeneratesHash, newSourcesHash, nil
}

func (checker *ChecksumChecker) SetUpToDate(t *ast.Task, sourceHash string) error {
	if len(t.Sources) == 0 && len(t.Generates) == 0 {
		return nil
	}

	if checker.dry {
		return nil
	}

	newSourcesHash, err := checker.checksum(t, t.Sources)
	if err != nil {
		return err
	}

	checksumFile := checker.checksumFilePath(t)

	if sourceHash != "" && newSourcesHash != sourceHash {
		// sources have changed since the task was executed, remove the checksum file
		// since the next execution will have a different checksum
		os.Remove(checksumFile)
		return nil
	}

	newGeneratesHash, err := checker.checksum(t, t.Generates)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepathext.SmartJoin(checker.tempDir, "checksum"), 0o755)
	if err = os.WriteFile(checksumFile, []byte(newSourcesHash+"\n"+newGeneratesHash+"\n"), 0o644); err != nil {
		return err
	}

	return nil
}

func (checker *ChecksumChecker) Value(t *ast.Task) (any, error) {
	c1, err := checker.checksum(t, t.Sources)
	if err != nil {
		return c1, err
	}
	c2, err := checker.checksum(t, t.Generates)
	return c1 + "\n" + c2, err
}

func (checker *ChecksumChecker) OnError(t *ast.Task) error {
	if len(t.Sources) == 0 {
		return nil
	}
	return os.Remove(checker.checksumFilePath(t))
}

func (*ChecksumChecker) Kind() string {
	return "checksum"
}

func (c *ChecksumChecker) checksum(t *ast.Task, globs []*ast.Glob) (string, error) {
	sources, err := Globs(t.Dir, globs)
	if err != nil {
		return "", err
	}

	h := xxh3.New()
	buf := make([]byte, 128*1024)
	for _, f := range sources {
		// also sum the filename, so checksum changes for renaming a file
		if rel, err := filepath.Rel(t.Dir, f); err == nil {
			h.WriteString(rel)
		} else {
			// couldn't make a relative path, use the full path to be safe
			h.WriteString(f)
		}
		// if we have a symlink here: we hash the link and *not* the target content
		if fi, err := os.Stat(f); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(f)
			if err != nil {
				return "", err
			}
			h.WriteString(link)
		} else {
			f, err := os.Open(f)
			if err != nil {
				return "", err
			}
			_, err = io.CopyBuffer(h, f, buf)
			f.Close()
			if err != nil {
				return "", err
			}
		}
	}
	hash := h.Sum128()
	return fmt.Sprintf("%x%x", hash.Hi, hash.Lo), nil
}

func (checker *ChecksumChecker) checksumFilePath(t *ast.Task) string {
	return filepath.Join(checker.tempDir, "checksum", normalizeFilename(t.Name()))
}

var checksumFilenameRegexp = regexp.MustCompile("[^A-z0-9]")

// replaces invalid characters on filenames with "-"
func normalizeFilename(f string) string {
	return checksumFilenameRegexp.ReplaceAllString(f, "-")
}
