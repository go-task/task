package completion

import (
	"os"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/orderedmap"
	"github.com/go-task/task/v3/taskfile"
)

var (
	tasks = taskfile.Tasks{
		OrderedMap: orderedmap.FromMapWithOrder(
			map[string]*taskfile.Task{
				"foo": {
					Task: "foo",
					Cmds: []*taskfile.Cmd{
						{Cmd: "echo foo"},
					},
					Desc:    "Prints foo",
					Aliases: []string{"f"},
				},
				"bar": {
					Task: "bar",
					Cmds: []*taskfile.Cmd{
						{Cmd: "echo bar"},
					},
					Desc:    "Prints bar",
					Aliases: []string{"b"},
				},
			},
			[]string{"foo", "bar"},
		),
	}
	flags struct {
		foo    string
		bar    int
		noDesc bool
	}
)

func init() {
	os.Args[0] = "task"
	pflag.StringVarP(&flags.foo, "foo", "f", "default", "A regular flag")
	pflag.IntVar(&flags.bar, "bar", 99, "A flag with no short variant")
	pflag.BoolVar(&flags.noDesc, "no-desc", true, "")
}

func TestCompile(t *testing.T) {
	tests := []struct {
		shell string
	}{
		{shell: "bash"},
		{shell: "fish"},
		{shell: "powershell"},
		{shell: "zsh"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			completion, err := Compile(tt.shell, tasks)
			require.NoError(t, err)

			g := goldie.New(t, goldie.WithTestNameForDir(true))
			g.Assert(t, tt.shell, []byte(completion))
		})
	}
}
