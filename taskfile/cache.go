package taskfile

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultCacheTTL is one day (24 hours).
const DefaultCacheTTL = time.Duration(time.Hour * 24)

type Cache struct {
	dir string
	ttl time.Duration
}

// ErrExpired is returned when a cached file has expired.
var ErrExpired = errors.New("task: cache expired")

type CacheOption func(*Cache)

func NewCache(dir string, opts ...CacheOption) (*Cache, error) {
	dir = filepath.Join(dir, "remote")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	cache := &Cache{
		dir: dir,
		ttl: DefaultCacheTTL,
	}

	// Apply options.
	for _, opt := range opts {
		opt(cache)
	}

	return cache, nil
}

// WithTTL will override the default TTL setting on a new Cache.
func WithTTL(ttl time.Duration) CacheOption {
	return func(cache *Cache) {
		cache.ttl = ttl
	}
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
	fi, err := os.Stat(c.cacheFilePath(node))
	if err != nil {
		return nil, fmt.Errorf("could not stat cached file: %w", err)
	}

	expiresAt := fi.ModTime().Add(c.ttl)
	if expiresAt.Before(time.Now()) {
		return nil, ErrExpired
	}

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
	return c.filePath(node, "yaml")
}

func (c *Cache) checksumFilePath(node Node) string {
	return c.filePath(node, "checksum")
}

func (c *Cache) filePath(node Node, suffix string) string {
	lastDir, filename := node.FilenameAndLastDir()
	prefix := filename
	// Means it's not "", nor "." nor "/", so it's a valid directory
	if len(lastDir) > 1 {
		prefix = fmt.Sprintf("%s-%s", lastDir, filename)
	}
	return filepath.Join(c.dir, fmt.Sprintf("%s.%s.%s", prefix, c.key(node), suffix))
}

func (c *Cache) Clear() error {
	return os.RemoveAll(c.dir)
}
