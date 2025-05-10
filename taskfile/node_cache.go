package taskfile

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const remoteCacheDir = "remote"

type CacheNode struct {
	*baseNode
	source RemoteNode
}

func NewCacheNode(source RemoteNode, dir string) *CacheNode {
	return &CacheNode{
		baseNode: &baseNode{
			dir: filepath.Join(dir, remoteCacheDir),
		},
		source: source,
	}
}

func (node *CacheNode) Read() ([]byte, error) {
	return os.ReadFile(node.Location())
}

func (node *CacheNode) Write(data []byte) error {
	if err := node.CreateCacheDir(); err != nil {
		return err
	}
	return os.WriteFile(node.Location(), data, 0o644)
}

func (node *CacheNode) ReadTimestamp() time.Time {
	b, err := os.ReadFile(node.timestampPath())
	if err != nil {
		return time.Time{}.UTC()
	}
	timestamp, err := time.Parse(time.RFC3339, string(b))
	if err != nil {
		return time.Time{}.UTC()
	}
	return timestamp.UTC()
}

func (node *CacheNode) WriteTimestamp(t time.Time) error {
	if err := node.CreateCacheDir(); err != nil {
		return err
	}
	return os.WriteFile(node.timestampPath(), []byte(t.Format(time.RFC3339)), 0o644)
}

func (node *CacheNode) ReadChecksum() string {
	b, _ := os.ReadFile(node.checksumPath())
	return string(b)
}

func (node *CacheNode) WriteChecksum(checksum string) error {
	if err := node.CreateCacheDir(); err != nil {
		return err
	}
	return os.WriteFile(node.checksumPath(), []byte(checksum), 0o644)
}

func (node *CacheNode) CreateCacheDir() error {
	if err := os.MkdirAll(node.dir, 0o755); err != nil {
		return err
	}
	return nil
}

func (node *CacheNode) ChecksumPrompt(checksum string) string {
	cachedChecksum := node.ReadChecksum()
	switch {

	// If the checksum doesn't exist, prompt the user to continue
	case cachedChecksum == "":
		return taskfileUntrustedPrompt

	// If there is a cached hash, but it doesn't match the expected hash, prompt the user to continue
	case cachedChecksum != checksum:
		return taskfileChangedPrompt

	default:
		return ""
	}
}

func (node *CacheNode) Location() string {
	return node.filePath("yaml")
}

func (node *CacheNode) checksumPath() string {
	return node.filePath("checksum")
}

func (node *CacheNode) timestampPath() string {
	return node.filePath("timestamp")
}

func (node *CacheNode) filePath(suffix string) string {
	return filepath.Join(node.dir, fmt.Sprintf("%s.%s", node.source.CacheKey(), suffix))
}

func checksum(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
