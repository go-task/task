package taskfunc_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/go-sprout/sprout/pesticide"
	taskfunc "github.com/go-task/task/v3/internal/templater/taskfunc"
)

func TestNumCPUs(t *testing.T) {
	expected := runtime.NumCPU()

	tc := []pesticide.TestCase{
		{Input: `{{ numCPU }}`, ExpectedOutput: fmt.Sprint(expected)},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestCatLines(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ catLines "a\nb" }}`, ExpectedOutput: "a b"},
		{Input: `{{ catLines "a\r\nb" }}`, ExpectedOutput: "a b"},
		{Input: `{{ catLines "a\nb\n" }}`, ExpectedOutput: "a b "},
		{Input: `{{ catLines "a\nb\r\n" }}`, ExpectedOutput: "a b "},
		{Input: `{{ catLines "a\r\nb\n\n" }}`, ExpectedOutput: "a b  "},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestSplitLines(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ splitLines "a\nb" }}`, ExpectedOutput: "[a b]"},
		{Input: `{{ splitLines "a\r\nb" }}`, ExpectedOutput: "[a b]"},
		{Input: `{{ splitLines "a\nb\n" }}`, ExpectedOutput: "[a b ]"},
		{Input: `{{ splitLines "a\nb\r\n" }}`, ExpectedOutput: "[a b ]"},
		{Input: `{{ splitLines "a\r\nb\n\n" }}`, ExpectedOutput: "[a b  ]"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestShellQuote(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ shellQuote "a b" }}`, ExpectedOutput: "'a b'"},
		{Input: `{{ shellQuote "a b c" }}`, ExpectedOutput: "'a b c'"},
		{Input: `{{ shellQuote "a'b" }}`, ExpectedOutput: "\"a'b\""},
		{Input: `{{ shellQuote "a\"b" }}`, ExpectedOutput: "'a\"b'"},
		{Name: "TestAlias", Input: `{{ q "a b" }}`, ExpectedOutput: "'a b'"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestSplitArgs(t *testing.T) {
	tc := []pesticide.TestCase{
		{Input: `{{ splitArgs "a b" }}`, ExpectedOutput: "[a b]"},
		{Input: `{{ splitArgs "a b c" }}`, ExpectedOutput: "[a b c]"},
		{Input: `{{ splitArgs "ab" }}`, ExpectedOutput: "[ab]"},
		{Name: "TestSpaceArg", Input: `{{ splitArgs "'a b' c" | join "-" }}`, ExpectedOutput: "a b-c"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestMergeArgs(t *testing.T) {
	var dest map[string]any

	tc := []pesticide.TestCase{
		{Name: "TestEmpty", Input: `{{merge .}}`, ExpectedOutput: "map[]"},
		{Name: "TestWithOneMap", Input: `{{merge .}}`, ExpectedOutput: "map[a:1 b:2]", Data: map[string]any{"a": 1, "b": 2}},
		{Name: "TestWithTwoMaps", Input: `{{merge .Dest .Src1}}`, ExpectedOutput: "map[a:1 b:2 c:3 d:4]", Data: map[string]any{"Dest": map[string]any{"a": 1, "b": 2}, "Src1": map[string]any{"c": 3, "d": 4}}},
		{Name: "TestWithOverwrite", Input: `{{merge .Dest .Src1}}`, ExpectedOutput: "map[a:3 b:2 d:4]", Data: map[string]any{"Dest": map[string]any{"a": 1, "b": 2}, "Src1": map[string]any{"a": 3, "d": 4}}},
		{Name: "TestWithZeroValues", Input: `{{merge .Dest .Src1}}`, ExpectedOutput: "map[a:2 b:true c:3 d:4]", Data: map[string]any{"Dest": map[string]any{"a": 0, "b": false}, "Src1": map[string]any{"a": 2, "b": true, "c": 3, "d": 4}}},
		{Name: "TestWithNotEnoughArgs", Input: `{{merge .}}`, ExpectedOutput: "map[a:1]", Data: map[string]any{"a": 1}},
		{Name: "TestWithDestNonInitialized", Input: `{{merge .A .B}}`, ExpectedOutput: "map[b:2]", Data: map[string]any{"A": dest, "B": map[string]any{"b": 2}}},
		{Name: "TestWithDestNotMap", Input: `{{merge .A .B}}`, Data: map[string]any{"A": 1, "B": map[string]any{"b": 2}}, ExpectedErr: "wrong type for value"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}

func TestSpew(t *testing.T) {

	dict := map[string]any{"a": 1}
	spewDictOutput := `(map[string]interface {}) (len=1) {
 (string) (len=1) "a": (int) 1
}
`
	list := []any{1}
	spewListOutput := `([]interface {}) (len=1 cap=1) {
 (int) 1
}
`
	strukt := struct{ A int }{1}
	spewStruktOutput := `(struct { A int }) {
 A: (int) 1
}
`

	tc := []pesticide.TestCase{
		{Input: `{{ spew .V }}`, ExpectedOutput: spewDictOutput, Data: map[string]any{"V": dict}},
		{Input: `{{ spew .V }}`, ExpectedOutput: spewListOutput, Data: map[string]any{"V": list}},
		{Input: `{{ spew .V }}`, ExpectedOutput: spewStruktOutput, Data: map[string]any{"V": strukt}},
		{Input: `{{ spew .V }}`, ExpectedOutput: "(interface {}) <nil>\n"},
	}

	pesticide.RunTestCases(t, taskfunc.NewRegistry(), tc)
}
