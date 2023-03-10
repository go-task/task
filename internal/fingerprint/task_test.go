package fingerprint

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile"
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
func TestIsTaskUpToDate(t *testing.T) {
	tests := []struct {
		name                    string
		task                    *taskfile.Task
		setupMockStatusChecker  func(m *MockStatusCheckable)
		setupMockSourcesChecker func(m *MockSourcesCheckable)
		expected                bool
	}{
		{
			name: "expect FALSE when no status or sources are defined",
			task: &taskfile.Task{
				Status:  nil,
				Sources: nil,
			},
			setupMockStatusChecker:  nil,
			setupMockSourcesChecker: nil,
			expected:                false,
		},
		{
			name: "expect TRUE when no status is defined and sources are up-to-date",
			task: &taskfile.Task{
				Status:  nil,
				Sources: []string{"sources"},
			},
			setupMockStatusChecker: nil,
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any()).Return(true, nil)
			},
			expected: true,
		},
		{
			name: "expect FALSE when no status is defined and sources are NOT up-to-date",
			task: &taskfile.Task{
				Status:  nil,
				Sources: []string{"sources"},
			},
			setupMockStatusChecker: nil,
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any()).Return(false, nil)
			},
			expected: false,
		},
		{
			name: "expect TRUE when status is up-to-date and sources are not defined",
			task: &taskfile.Task{
				Status:  []string{"status"},
				Sources: nil,
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any(), gomock.Any()).Return(true, nil)
			},
			setupMockSourcesChecker: nil,
			expected:                true,
		},
		{
			name: "expect TRUE when status and sources are up-to-date",
			task: &taskfile.Task{
				Status:  []string{"status"},
				Sources: []string{"sources"},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any(), gomock.Any()).Return(true, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any()).Return(true, nil)
			},
			expected: true,
		},
		{
			name: "expect FALSE when status is up-to-date, but sources are NOT up-to-date",
			task: &taskfile.Task{
				Status:  []string{"status"},
				Sources: []string{"sources"},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any(), gomock.Any()).Return(true, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any()).Return(false, nil)
			},
			expected: false,
		},
		{
			name: "expect FALSE when status is NOT up-to-date and sources are not defined",
			task: &taskfile.Task{
				Status:  []string{"status"},
				Sources: nil,
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			setupMockSourcesChecker: nil,
			expected:                false,
		},
		{
			name: "expect FALSE when status is NOT up-to-date, but sources are up-to-date",
			task: &taskfile.Task{
				Status:  []string{"status"},
				Sources: []string{"sources"},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any()).Return(true, nil)
			},
			expected: false,
		},
		{
			name: "expect FALSE when status and sources are NOT up-to-date",
			task: &taskfile.Task{
				Status:  []string{"status"},
				Sources: []string{"sources"},
			},
			setupMockStatusChecker: func(m *MockStatusCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			setupMockSourcesChecker: func(m *MockSourcesCheckable) {
				m.EXPECT().IsUpToDate(gomock.Any()).Return(false, nil)
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockStatusChecker := NewMockStatusCheckable(ctrl)
			if tt.setupMockStatusChecker != nil {
				tt.setupMockStatusChecker(mockStatusChecker)
			}

			mockSourcesChecker := NewMockSourcesCheckable(ctrl)
			if tt.setupMockSourcesChecker != nil {
				tt.setupMockSourcesChecker(mockSourcesChecker)
			}

			result, err := IsTaskUpToDate(
				context.Background(),
				tt.task,
				WithStatusChecker(mockStatusChecker),
				WithSourcesChecker(mockSourcesChecker),
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
