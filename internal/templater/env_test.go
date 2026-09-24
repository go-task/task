package templater

import (
	"os"
	"testing"
)

func TestEnvFuncPrefersProcessEnv(t *testing.T) {
	t.Parallel()

	key := "TASK_TEST_ENV_FUNC_PROCESS"
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	if err := os.Setenv(key, "from-process"); err != nil {
		t.Fatal(err)
	}

	fn := envFunc(map[string]any{key: "from-data"})
	if got := fn(key); got != "from-process" {
		t.Fatalf("env(%q) = %q; want from-process", key, got)
	}
}

func TestEnvFuncFallsBackToTemplateData(t *testing.T) {
	t.Parallel()

	key := "TASK_TEST_ENV_FUNC_DOTENV"
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	_ = os.Unsetenv(key)

	fn := envFunc(map[string]any{key: "from-dotenv"})
	if got := fn(key); got != "from-dotenv" {
		t.Fatalf("env(%q) = %q; want from-dotenv", key, got)
	}
}

func TestEnvFuncMissingReturnsEmpty(t *testing.T) {
	t.Parallel()

	key := "TASK_TEST_ENV_FUNC_MISSING"
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	_ = os.Unsetenv(key)

	fn := envFunc(map[string]any{})
	if got := fn(key); got != "" {
		t.Fatalf("env(%q) = %q; want empty", key, got)
	}
}
