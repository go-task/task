package templater

import (
	stdos "os"
	"testing"
)

func TestJoinEnv(t *testing.T) {
	t.Parallel()

	got := joinEnv("/tmp/tools/bin", "/usr/local/bin", "/usr/bin")
	want := "/tmp/tools/bin" + string(stdos.PathListSeparator) + "/usr/local/bin" + string(stdos.PathListSeparator) + "/usr/bin"
	if got != want {
		t.Fatalf("joinEnv() = %q, want %q", got, want)
	}
}

func TestJoinEnvTemplateFunc(t *testing.T) {
	t.Parallel()

	joinEnvFunc, ok := templateFuncs["joinEnv"].(func(...string) string)
	if !ok {
		t.Fatalf("joinEnv template function has unexpected type: %T", templateFuncs["joinEnv"])
	}

	got := joinEnvFunc("a", "b")
	want := "a" + string(stdos.PathListSeparator) + "b"
	if got != want {
		t.Fatalf("template joinEnv() = %q, want %q", got, want)
	}
}
