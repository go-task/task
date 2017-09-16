package status

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Checksum validades if a task is up to date by calculating its source
// files checksum
type Checksum struct {
	Dir     string
	Task    string
	Sources []string
}

// IsUpToDate implements the Checker interface
func (c *Checksum) IsUpToDate() (bool, error) {
	checksumFile := filepath.Join(c.Dir, ".task", c.Task)

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

	_ = os.MkdirAll(filepath.Join(c.Dir, ".task"), 0755)
	if err = ioutil.WriteFile(checksumFile, []byte(newMd5), 0644); err != nil {
		return false, err
	}
	return oldMd5 == newMd5, nil
}

func (c *Checksum) checksum(files ...string) (string, error) {
	h := md5.New()

	for _, f := range files {
		f, err := os.Open(f)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// OnError implements the Checker interface
func (c *Checksum) OnError() error {
	return os.Remove(filepath.Join(c.Dir, ".task", c.Task))
}
