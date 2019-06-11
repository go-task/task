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
	Dir     string
	Task    string
	Sources []string
	Dry     bool
}

// IsUpToDate implements the Checker interface
func (c *Checksum) IsUpToDate() (bool, error) {
	checksumFile := c.checksumFilePath()

	data, _ := ioutil.ReadFile(checksumFile)
	oldMd5 := strings.TrimSpace(string(data))

	sources, err := glob(c.Dir, c.Sources)
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
	return oldMd5 == newMd5, nil
}

func (t *Checksum) Kind() string {
	return "checksum"
}

func (c *Checksum) checksum(files ...string) (string, error) {
	h := md5.New()

	for _, f := range files {
		f, err := os.Open(f)
		if err != nil {
			return "", err
		}
		info, err := f.Stat()
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			continue
		}
		// also sum the filename, so checksum changes for renaming a file
		if _, err = io.Copy(h, strings.NewReader(info.Name())); err != nil {
			return "", err
		}
		if _, err = io.Copy(h, f); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Value implements the Chcker Interface
func (c *Checksum) Value() (string, error) {
	return c.checksum()
}

// OnError implements the Checker interface
func (c *Checksum) OnError() error {
	return os.Remove(c.checksumFilePath())
}

func (c *Checksum) checksumFilePath() string {
	return filepath.Join(c.Dir, ".task", "checksum", c.normalizeFilename(c.Task))
}

var checksumFilenameRegexp = regexp.MustCompile("[^A-z0-9]")

// replaces invalid caracters on filenames with "-"
func (*Checksum) normalizeFilename(f string) string {
	return checksumFilenameRegexp.ReplaceAllString(f, "-")
}
