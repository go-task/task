package taskfunc_test

import (
	"testing"

	"github.com/go-sprout/sprout/pesticide"
	taskfunc "github.com/go-task/task/v3/internal/templater/taskfunc"
)

func TestOs(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ os }}`, ExpectedOutput: "linux"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestArch(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ arch }}`, ExpectedOutput: "amd64"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestFromSlash(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ fromSlash "a/b" }}`, ExpectedOutput: "a/b"},
		{Input: `{{ fromSlash "a\\b" }}`, ExpectedOutput: "a\\b"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestToSlash(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ toSlash "a\\b" }}`, ExpectedOutput: "a\\b"},
		{Input: `{{ toSlash "a/b" }}`, ExpectedOutput: "a/b"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestExeExt(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ exeExt }}`, ExpectedOutput: ""},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestJoinPath(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ joinPath "a" "b" }}`, ExpectedOutput: "a/b"},
		{Input: `{{ joinPath "a/b" "c" }}`, ExpectedOutput: "a/b/c"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestRelPath(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ relPath "/a" "/a/b/c" }}`, ExpectedOutput: "b/c"},
		{Input: `{{ relPath "\\a" "\\b\\c" }}`, ExpectedOutput: "../\\b\\c"},
		{Input: `{{ relPath "/a/b" "/a/b/c" }}`, ExpectedOutput: "c"},
		{Input: `{{ relPath "/a" "./b/c" }}`, ExpectedErr: "relPath: Rel: can't make ./b/c relative to /a"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}
