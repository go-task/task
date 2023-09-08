package read

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Cache struct {
	dir string
}

func NewCache(dir string) (*Cache, error) {
	dir = filepath.Join(dir, "remote")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{
		dir: dir,
	}, nil
}

func checksum(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *Cache) write(node Node, b []byte) error {
	return os.WriteFile(c.cacheFilePath(node), b, 0o644)
}

func (c *Cache) read(node Node) ([]byte, error) {
	return os.ReadFile(c.cacheFilePath(node))
}

func (c *Cache) writeChecksum(node Node, checksum string) error {
	return os.WriteFile(c.checksumFilePath(node), []byte(checksum), 0o644)
}

func (c *Cache) readChecksum(node Node) string {
	b, _ := os.ReadFile(c.checksumFilePath(node))
	return string(b)
}

func (c *Cache) key(node Node) string {
	return strings.TrimRight(checksum([]byte(node.Location())), "=")
}

func (c *Cache) cacheFilePath(node Node) string {
	return filepath.Join(c.dir, fmt.Sprintf("%s.yaml", c.key(node)))
}

func (c *Cache) checksumFilePath(node Node) string {
	return filepath.Join(c.dir, fmt.Sprintf("%s.checksum", c.key(node)))
}
