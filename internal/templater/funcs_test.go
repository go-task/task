package templater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathCleansRelativeSegments(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "foo", "..", "bar")

	resolved, err := resolvePath(input)
	if err != nil {
		t.Fatalf("resolvePath returned error: %v", err)
	}

	expected, err := filepath.Abs(filepath.Join(tmpDir, "bar"))
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}

	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}

func TestResolvePathResolvesSymlink(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	link := filepath.Join(tmpDir, "link")
	if err := os.Symlink(targetDir, link); err != nil {
		t.Skipf("symlinks not supported in this environment: %v", err)
	}

	resolved, err := resolvePath(filepath.Join(link, "..", "link"))
	if err != nil {
		t.Fatalf("resolvePath returned error: %v", err)
	}

	expected, err := filepath.Abs(targetDir)
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}

	if resolved != expected {
		t.Fatalf("expected symlink to resolve to %q, got %q", expected, resolved)
	}
}
