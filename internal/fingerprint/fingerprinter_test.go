package fingerprint

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

// TruthTable
//
// | Status up-to-date | Sources up-to-date | Task is up-to-date |
// | ----------------- | ------------------ | ------------------ |
// | not set           | not set            | false              |
// | not set           | true               | true               |
// | not set           | false              | false              |
// | true              | not set            | true               |
// | true              | true               | true               |
// | true              | false              | false              |
// | false             | not set            | false              |
// | false             | true               | false              |
// | false             | false              | false              |
func TestFingerprinterUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		task                    *ast.Task
		setupMockStatusChecker  func(m *MockStatusCheckable)
		setupMockSourcesChecker func(m *MockSourcesCheckable)
		expected                bool
	}{
		{
			name: "expect FALSE when no status or sources are defined",
			task: &ast.Task{
				Status:  nil,
				Sources: nil,
			},
			setupMockStatusChecker:  nil,
			setupMockSourcesChecker: nil,
			expected:                false,
		},
		{
			name: "expect TRUE when no status is defined and sources are up-to-date",
			task: &ast.Task{
				Status:  nil,
				Sources: []*ast.Glob{{Glob: "sources"}},
			},
			setupMockStatusChecker: nil,
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything).Return(true, nil)
			},
			expected: true,
		},
		{
			name: "expect FALSE when no status is defined and sources are NOT up-to-date",
			task: &ast.Task{
				Status:  nil,
				Sources: []*ast.Glob{{Glob: "sources"}},
			},
			setupMockStatusChecker: nil,
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything).Return(false, nil)
			},
			expected: false,
		},
		{
			name: "expect TRUE when status is up-to-date and sources are not defined",
			task: &ast.Task{
				Status:  []string{"status"},
				Sources: nil,
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything, mock.Anything).Return(true, nil)
			},
			setupMockSourcesChecker: nil,
			expected:                true,
		},
		{
			name: "expect TRUE when status and sources are up-to-date",
			task: &ast.Task{
				Status:  []string{"status"},
				Sources: []*ast.Glob{{Glob: "sources"}},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything, mock.Anything).Return(true, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything).Return(true, nil)
			},
			expected: true,
		},
		{
			name: "expect FALSE when status is up-to-date, but sources are NOT up-to-date",
			task: &ast.Task{
				Status:  []string{"status"},
				Sources: []*ast.Glob{{Glob: "sources"}},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything, mock.Anything).Return(true, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything).Return(false, nil)
			},
			expected: false,
		},
		{
			name: "expect FALSE when status is NOT up-to-date and sources are not defined",
			task: &ast.Task{
				Status:  []string{"status"},
				Sources: nil,
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything, mock.Anything).Return(false, nil)
			},
			setupMockSourcesChecker: nil,
			expected:                false,
		},
		{
			name: "expect FALSE when status is NOT up-to-date, but sources are up-to-date",
			task: &ast.Task{
				Status:  []string{"status"},
				Sources: []*ast.Glob{{Glob: "sources"}},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything, mock.Anything).Return(false, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything).Return(true, nil)
			},
			expected: false,
		},
		{
			name: "expect FALSE when status and sources are NOT up-to-date",
			task: &ast.Task{
				Status:  []string{"status"},
				Sources: []*ast.Glob{{Glob: "sources"}},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything, mock.Anything).Return(false, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(mock.Anything).Return(false, nil)
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockStatusChecker := NewMockStatusCheckable(t)
			if tt.setupMockStatusChecker != nil {
				tt.setupMockStatusChecker(mockStatusChecker)
			}

			mockSourcesChecker := NewMockSourcesCheckable(t)
			if tt.setupMockSourcesChecker != nil {
				tt.setupMockSourcesChecker(mockSourcesChecker)
			}

			f := NewFingerprinter("checksum", "", false, nil,
				WithStatusChecker(mockStatusChecker),
				WithSourcesChecker(mockSourcesChecker),
			)
			result, err := f.UpToDate(t.Context(), tt.task)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// The task's own method wins over the Taskfile default, for the injected
// variable as much as for the up-to-date check.
func TestFingerprinterMethodResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		defaultMethod string
		method        string
		expectedKind  string
		expectedValue any
	}{
		{
			name:          "task method wins over the default",
			defaultMethod: "checksum",
			method:        "timestamp",
			expectedKind:  "timestamp",
			expectedValue: time.Time{},
		},
		{
			name:          "default method is inherited when the task declares none",
			defaultMethod: "timestamp",
			expectedKind:  "timestamp",
			expectedValue: time.Time{},
		},
		{
			name:          "checksum is inherited too",
			defaultMethod: "checksum",
			expectedKind:  "checksum",
			expectedValue: "",
		},
		{
			name:          "none is inherited too",
			defaultMethod: "none",
			expectedKind:  "none",
			expectedValue: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "source.txt"), []byte("content"), 0o644))
			task := &ast.Task{
				Dir:     dir,
				Method:  tt.method,
				Sources: []*ast.Glob{{Glob: "source.txt"}},
			}

			f := NewFingerprinter(tt.defaultMethod, t.TempDir(), true, nil)

			assert.Equal(t, tt.expectedKind, f.Kind(task))

			// A timestamp checker yields a time, the other two a string.
			value, err := f.SourceValue(task)
			require.NoError(t, err)
			assert.IsType(t, tt.expectedValue, value)
		})
	}
}

// Only the entry points that need a checker reject an invalid method; Kind
// tolerates it, so that --force runs still compile.
func TestFingerprinterInvalidMethod(t *testing.T) {
	t.Parallel()

	const wantErr = `task: invalid method "Checksum"`
	task := &ast.Task{Sources: []*ast.Glob{{Glob: "source.txt"}}}
	f := NewFingerprinter("Checksum", t.TempDir(), true, nil)

	assert.Equal(t, "checksum", f.Kind(task))

	_, err := f.SourceValue(task)
	require.EqualError(t, err, wantErr)
	_, err = f.UpToDate(t.Context(), task)
	require.EqualError(t, err, wantErr)
	require.EqualError(t, f.OnError(task), wantErr)
}
