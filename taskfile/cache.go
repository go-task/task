package taskfile

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
)

type Cache struct {
	dir string
}

type metadata struct {
	Checksum     string
	TaskfileName string
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
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func checksumSource(s source) (string, error) {
	h := sha256.New()

	entries, err := os.ReadDir(s.FileDirectory)
	if err != nil {
		return "", fmt.Errorf("could not list files at %s: %w", s.FileDirectory, err)
	}

	for _, e := range entries {
		if e.Type().IsRegular() {
			path := filepath.Join(s.FileDirectory, e.Name())
			f, err := os.Open(path)
			if err != nil {
				return "", fmt.Errorf("error opening file %s for checksumming: %w", path, err)
			}
			if _, err := f.WriteTo(h); err != nil {
				f.Close()
				return "", fmt.Errorf("error reading file %s for checksumming: %w", path, err)
			}
			f.Close()
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

func (c *Cache) write(node Node, src source) (*source, error) {
	// Clear metadata file so that if the rest of the operations fail part-way we don't
	// end up in an inconsistent state where we've written the contents but have old metadata
	if err := c.clearMetadata(node); err != nil {
		return nil, err
	}

	p, err := c.contentsPath(node)
	if err != nil {
		return nil, err
	}

	switch fi, err := os.Stat(p); {
	case errors.Is(err, os.ErrNotExist):
		// Nothign to clear, do nothing

	case !fi.IsDir():
		return nil, fmt.Errorf("error writing to contents path %s: not a directory", p)

	case err != nil:
		return nil, fmt.Errorf("error cheacking for previous contents path %s: %w", p, err)

	default:
		err := os.RemoveAll(p)
		if err != nil {
			return nil, fmt.Errorf("error clearing contents directory: %s", err)
		}
	}

	if err := os.Rename(src.FileDirectory, p); err != nil {
		return nil, err
	}

	// TODO Clean up
	src.FileDirectory = p

	cs, err := checksumSource(src)
	if err != nil {
		return nil, err
	}

	m := metadata{
		Checksum:     cs,
		TaskfileName: src.Filename,
	}

	if err := c.storeMetadata(node, m); err != nil {
		return nil, fmt.Errorf("error storing metadata for node %s: %w", node.Location(), err)
	}

	return &src, nil
}

func (c *Cache) read(node Node) (*source, error) {
	path, err := c.contentsPath(node)
	if err != nil {
		return nil, err
	}

	m, err := c.readMetadata(node)
	if err != nil {
		return nil, err
	}

	taskfileName := m.TaskfileName

	content, err := os.ReadFile(filepath.Join(path, m.TaskfileName))
	if err != nil {
		return nil, err
	}

	return &source{
		FileContent:   content,
		FileDirectory: path,
		Filename:      taskfileName,
	}, nil
}

func (c *Cache) readChecksum(node Node) string {
	m, err := c.readMetadata(node)
	if err != nil {
		return ""
	}
	return m.Checksum
}

func (c *Cache) clearMetadata(node Node) error {
	path, err := c.metadataFilePath(node)
	if err != nil {
		return fmt.Errorf("error clearing metadata file at %s: %w", path, err)
	}

	fi, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if !fi.Mode().IsRegular() {
		return fmt.Errorf("path is not a real file when trying to delete metadata file: %s", path)
	}

	// if err := os.Remove(path)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("error removing metadata file %s: %w", path, err)
	}

	return nil
}

func (c *Cache) storeMetadata(node Node, m metadata) error {
	path, err := c.metadataFilePath(node)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("error creating metadata file %s: %w", path, err)
	}
	defer f.Close()

	if err := yaml.NewEncoder(f).Encode(m); err != nil {
		return fmt.Errorf("error writing metadata into %s: %w", path, err)
	}

	return nil
}

func (c *Cache) readMetadata(node Node) (*metadata, error) {
	path, err := c.metadataFilePath(node)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening metadata file %s: %w", path, err)
	}
	defer f.Close()

	var m *metadata
	if err := yaml.NewDecoder(f).Decode(&m); err != nil {
		return nil, fmt.Errorf("error reading metadata file %s: %w", path, err)
	}

	return m, nil
}

func (c *Cache) key(node Node) string {
	return strings.TrimRight(checksum([]byte(node.Location())), "=")
}

func (c *Cache) contentsPath(node Node) (string, error) {
	return c.cacheFilePath(node, "contents")
}

func (c *Cache) metadataFilePath(node Node) (string, error) {
	return c.cacheFilePath(node, "metadata.yaml")
}

func (c *Cache) cacheFilePath(node Node, filename string) (string, error) {
	lastDir, prefix := node.FilenameAndLastDir()
	// Means it's not "", nor "." nor "/", so it's a valid directory
	if len(lastDir) > 1 {
		prefix = fmt.Sprintf("%s-%s", lastDir, prefix)
	}

	dir := filepath.Join(c.dir, fmt.Sprintf("%s.%s", prefix, c.key(node)))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("error creating cache dir %s: %w", dir, err)
	}

	return filepath.Join(dir, filename), nil
}

func (c *Cache) Clear() error {
	return os.RemoveAll(c.dir)
}
