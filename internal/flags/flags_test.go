package flags

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	taskrcast "github.com/go-task/task/v3/taskrc/ast"
)

func TestGetConfigOutputEnvVars(t *testing.T) {
	t.Run("TASK_OUTPUT sets the output name", func(t *testing.T) {
		t.Setenv("TASK_OUTPUT", "group")
		got := getConfig[string](nil, "OUTPUT", func() *string { return nil }, "")
		require.Equal(t, "group", got)
	})

	t.Run("TASK_OUTPUT_GROUP_BEGIN sets the begin template", func(t *testing.T) {
		t.Setenv("TASK_OUTPUT_GROUP_BEGIN", "::group::{{.TASK}}")
		got := getConfig[string](nil, "OUTPUT_GROUP_BEGIN", func() *string { return nil }, "")
		require.Equal(t, "::group::{{.TASK}}", got)
	})

	t.Run("TASK_OUTPUT_GROUP_END sets the end template", func(t *testing.T) {
		t.Setenv("TASK_OUTPUT_GROUP_END", "::endgroup::")
		got := getConfig[string](nil, "OUTPUT_GROUP_END", func() *string { return nil }, "")
		require.Equal(t, "::endgroup::", got)
	})

	t.Run("TASK_OUTPUT_GROUP_ERROR_ONLY sets the error-only flag", func(t *testing.T) {
		t.Setenv("TASK_OUTPUT_GROUP_ERROR_ONLY", "true")
		got := getConfig[bool](nil, "OUTPUT_GROUP_ERROR_ONLY", func() *bool { return nil }, false)
		require.True(t, got)
	})

	t.Run("falls back to the default when the env var is unset", func(t *testing.T) {
		require.Equal(t, "", getConfig[string](nil, "OUTPUT", func() *string { return nil }, ""))
		require.False(t, getConfig[bool](nil, "OUTPUT_GROUP_ERROR_ONLY", func() *bool { return nil }, false))
	})

	t.Run("invalid bool env var falls back to the default", func(t *testing.T) {
		t.Setenv("TASK_OUTPUT_GROUP_ERROR_ONLY", "not-a-bool")
		require.False(t, getConfig[bool](nil, "OUTPUT_GROUP_ERROR_ONLY", func() *bool { return nil }, false))
	})
}

func TestGetConfigPrecedence(t *testing.T) {
	t.Run("env var takes precedence over the taskrc config", func(t *testing.T) {
		t.Setenv("TASK_VERBOSE", "true")
		config := &taskrcast.TaskRC{Verbose: boolPtr(false)}
		require.True(t, getConfig[bool](config, "VERBOSE", func() *bool { return config.Verbose }, false))
	})

	t.Run("taskrc config is used when the env var is unset", func(t *testing.T) {
		config := &taskrcast.TaskRC{Verbose: boolPtr(true)}
		require.True(t, getConfig[bool](config, "VERBOSE", func() *bool { return config.Verbose }, false))
	})

	t.Run("duration env vars are parsed", func(t *testing.T) {
		t.Setenv("TASK_REMOTE_TIMEOUT", "30s")
		got := getConfig[time.Duration](nil, "REMOTE_TIMEOUT", func() *time.Duration { return nil }, time.Second*10)
		require.Equal(t, 30*time.Second, got)
	})
}

func boolPtr(b bool) *bool { return &b }
