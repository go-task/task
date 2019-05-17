package taskfile_test

import (
	"testing"

	"github.com/go-task/task/v2/internal/taskfile"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestPreconditionParse(t *testing.T) {
	tests := []struct {
		content  string
		v        interface{}
		expected interface{}
	}{
		{
			"test -f foo.txt",
			&taskfile.Precondition{},
			&taskfile.Precondition{Sh: `test -f foo.txt`, Msg: "`test -f foo.txt` failed", IgnoreError: false},
		},
		{
			"sh: '[ 1 = 0 ]'",
			&taskfile.Precondition{},
			&taskfile.Precondition{Sh: "[ 1 = 0 ]", Msg: "[ 1 = 0 ] failed", IgnoreError: false},
		},
		{`
sh: "[ 1 = 2 ]"
msg: "1 is not 2"
`,
			&taskfile.Precondition{},
			&taskfile.Precondition{Sh: "[ 1 = 2 ]", Msg: "1 is not 2", IgnoreError: false},
		},
		{`
sh: "[ 1 = 2 ]"
msg: "1 is not 2"
ignore_error: true
`,
			&taskfile.Precondition{},
			&taskfile.Precondition{Sh: "[ 1 = 2 ]", Msg: "1 is not 2", IgnoreError: true},
		},
	}
	for _, test := range tests {
		err := yaml.Unmarshal([]byte(test.content), test.v)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, test.v)
	}
}
