package status

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Checksum validades if a task is up to date by calculating its source
// files checksum
type Checksum struct {
	Dir       string
	Task      string
	Sources   []string
	Generates []string
	Dry       bool
}

// IsUpToDate implements the Checker interface
func (c *Checksum) IsUpToDate() (bool, error) {
	if len(c.Sources) == 0 {
		return false, nil
	}

	checksumFile := c.checksumFilePath()

	data, _ := ioutil.ReadFile(checksumFile)
	oldMd5 := strings.TrimSpace(string(data))

	sources, err := globs(c.Dir, c.Sources)
	if err != nil {
		return false, err
	}

	newMd5, err := c.checksum(sources...)
	if err != nil {
		return false, nil
	}

	if !c.Dry {
		_ = os.MkdirAll(filepath.Join(c.Dir, ".task", "checksum"), 0755)
		if err = ioutil.WriteFile(checksumFile, []byte(newMd5+"\n"), 0644); err != nil {
			return false, err
		}
	}

	if len(c.Generates) > 0 {
		// For each specified 'generates' field, check whether the files actually exist
		for _, g := range c.Generates {
			generates, err := glob(c.Dir, g)
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

func (c *Checksum) checksum(files ...string) (string, error) {
	h := md5.New()

	for _, f := range files {
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
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Value implements the Checker Interface
func (c *Checksum) Value() (interface{}, error) {
	return c.checksum()
}

// OnError implements the Checker interface
func (c *Checksum) OnError() error {
	return os.Remove(c.checksumFilePath())
}

// Kind implements the Checker Interface
func (*Checksum) Kind() string {
	return "checksum"
}

func (c *Checksum) checksumFilePath() string {
	return filepath.Join(c.Dir, ".task", "checksum", c.normalizeFilename(c.Task))
}

var checksumFilenameRegexp = regexp.MustCompile("[^A-z0-9]")

// replaces invalid caracters on filenames with "-"
func (*Checksum) normalizeFilename(f string) string {
	return checksumFilenameRegexp.ReplaceAllString(f, "-")
}
