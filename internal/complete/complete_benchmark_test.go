package complete_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-task/task/v3/internal/complete"
)

func BenchmarkComplete(b *testing.B) {
	var tasks strings.Builder
	tasks.WriteString("version: '3'\nvars:\n  TITLE: example\ntasks:\n")
	for i := range 1000 {
		fmt.Fprintf(&tasks, "  task%d:\n    desc: Static description\n", i)
	}
	const loop = `version: '3'
tasks:
  build:
    vars:
      ITEMS: '{{until 10000}}'
    cmds:
      - for: {var: ITEMS}
        cmd: 'echo {{.ITEM}}'
`
	for _, tc := range []struct {
		name     string
		taskfile string
		words    []string
	}{
		{"Static1000", tasks.String(), []string{""}},
		{"OneTemplatedDescription1000", strings.Replace(tasks.String(), "Static description", "'{{.TITLE}}'", 1), []string{""}},
		{"AfterLargeLoop", loop, []string{"build", ""}},
		{"StaticEnumWithLargeLoop", loop + "    requires:\n      vars:\n        - name: ENV\n          enum: [dev, prod]\n", []string{"build", ""}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			e := setupExecutorWith(b, tc.taskfile)
			fs := newTestFlagSet()
			b.ReportAllocs()
			for b.Loop() {
				complete.Complete(e, fs, tc.words, complete.Options{})
			}
		})
	}
}
