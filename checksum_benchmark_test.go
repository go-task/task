package task_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
)

const (
	issue2853ManySmallYAMLFileCount = 20_000
	issue2853SmallYAMLFileSize      = 5
	issue2853FewLargeYAMLFileCount  = 4
	issue2853LargeYAMLFileSize      = 128 * 1024 * 1024
)

func BenchmarkIssue2853ManySmallSparseYAMLFiles(b *testing.B) {
	dir := b.TempDir()
	createIssue2853Fixture(b, dir, issue2853ManySmallYAMLFileCount, issue2853SmallYAMLFileSize)

	benchmarkIssue2853Modes(b, dir, issue2853ManySmallYAMLFileCount, issue2853SmallYAMLFileSize)
}

func BenchmarkIssue2853FewLargeSparseYAMLFiles(b *testing.B) {
	dir := b.TempDir()
	createIssue2853Fixture(b, dir, issue2853FewLargeYAMLFileCount, issue2853LargeYAMLFileSize)

	benchmarkIssue2853Modes(b, dir, issue2853FewLargeYAMLFileCount, issue2853LargeYAMLFileSize)
}

func benchmarkIssue2853Modes(b *testing.B, dir string, fileCount int, fileSize int64) {
	b.Helper()

	for _, mode := range []struct {
		name        string
		task        string
		expectCache bool
	}{
		{name: "checksum", task: "checksum-yaml", expectCache: true},
		{name: "timestamp", task: "timestamp-yaml", expectCache: true},
		{name: "none", task: "uncached-yaml"},
	} {
		b.Run(mode.name, func(b *testing.B) {
			benchmarkIssue2853Task(b, dir, mode.task, mode.expectCache, fileCount, fileSize)
		})
	}
}

func benchmarkIssue2853Task(
	b *testing.B,
	dir string,
	taskName string,
	expectCache bool,
	fileCount int,
	fileSize int64,
) {
	b.Helper()

	tempDir := task.TempDir{
		Remote:      filepath.Join(dir, ".task"),
		Fingerprint: filepath.Join(dir, ".task"),
	}

	if expectCache {
		e := task.NewExecutor(
			task.WithDir(dir),
			task.WithStdout(io.Discard),
			task.WithStderr(io.Discard),
			task.WithTempDir(tempDir),
		)
		require.NoError(b, e.Setup())
		require.NoError(b, e.Run(b.Context(), &task.Call{Task: taskName}))
	}

	b.ReportAllocs()
	sourceBytes := int64(fileCount) * fileSize
	if expectCache {
		b.SetBytes(sourceBytes)
	}
	b.ResetTimer()
	for range b.N {
		var buff bytes.Buffer
		e := task.NewExecutor(
			task.WithDir(dir),
			task.WithStdout(&buff),
			task.WithStderr(&buff),
			task.WithTempDir(tempDir),
		)
		require.NoError(b, e.Setup())
		require.NoError(b, e.Run(b.Context(), &task.Call{Task: taskName}))
		if expectCache {
			require.Contains(b, buff.String(), fmt.Sprintf(`Task "%s" is up to date`, taskName))
		}
	}
	if expectCache {
		b.ReportMetric(float64(fileCount), "source_files/op")
		b.ReportMetric(float64(sourceBytes)/(1024*1024), "source_MiB/op")
	}
}

func createIssue2853Fixture(tb testing.TB, dir string, fileCount int, fileSize int64) {
	tb.Helper()

	taskfile := `version: '3'

tasks:
  checksum-yaml:
    sources:
      - path/to/folder/**/*.yaml
    generates:
      - out/checksum.txt
    cmds:
      - printf ok > out/checksum.txt

  timestamp-yaml:
    method: timestamp
    sources:
      - path/to/folder/**/*.yaml
    generates:
      - out/timestamp.txt
    cmds:
      - printf ok > out/timestamp.txt

  uncached-yaml:
    method: none
    cmds:
      - printf ok > out/uncached.txt
`
	require.NoError(tb, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644))
	require.NoError(tb, os.MkdirAll(filepath.Join(dir, "out"), 0o755))

	for i := 1; i <= fileCount; i++ {
		subdir := filepath.Join(dir, "path", "to", "folder", fmt.Sprintf("%04d", i/100))
		require.NoError(tb, os.MkdirAll(subdir, 0o755))
		name := filepath.Join(subdir, fmt.Sprintf("file-%05d.yaml", i))
		createSparseFile(tb, name, fileSize)
	}
}

func createSparseFile(tb testing.TB, name string, size int64) {
	tb.Helper()

	file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	require.NoError(tb, err)
	defer func() {
		require.NoError(tb, file.Close())
	}()
	require.NoError(tb, file.Truncate(size))
}
