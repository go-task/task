package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskReferencesFingerprintVar(t *testing.T) {
	t.Parallel()

	fingerprintRef := "{{.CHECKSUM}}"
	tests := []struct {
		name string
		task *Task
		want bool
	}{
		{
			name: "nil task",
			want: false,
		},
		{
			name: "status",
			task: &Task{Status: []string{"test -n " + fingerprintRef}},
			want: true,
		},
		{
			name: "command",
			task: &Task{Cmds: []*Cmd{{Cmd: "echo " + fingerprintRef}}},
			want: true,
		},
		{
			name: "task call",
			task: &Task{Cmds: []*Cmd{{Task: "build-" + fingerprintRef}}},
			want: true,
		},
		{
			name: "command condition",
			task: &Task{Cmds: []*Cmd{{If: "test -n " + fingerprintRef}}},
			want: true,
		},
		{
			name: "loop list nested value",
			task: &Task{Cmds: []*Cmd{{For: &For{List: []any{map[string]any{"value": fingerprintRef}}}}}},
			want: true,
		},
		{
			name: "loop matrix reference",
			task: &Task{Cmds: []*Cmd{{For: &For{Matrix: NewMatrix(
				&MatrixElement{Key: "item", Value: &MatrixRow{Ref: fingerprintRef}},
			)}}}},
			want: true,
		},
		{
			name: "command variable",
			task: &Task{Cmds: []*Cmd{{Vars: NewVars(
				&VarElement{Key: "VALUE", Value: Var{Value: []any{fingerprintRef}}},
			)}}},
			want: true,
		},
		{
			name: "dynamic command variable",
			task: &Task{Cmds: []*Cmd{{Vars: NewVars(
				&VarElement{Key: "VALUE", Value: Var{Sh: &fingerprintRef}},
			)}}},
			want: true,
		},
		{
			name: "dependency",
			task: &Task{Deps: []*Dep{{Task: "build-" + fingerprintRef}}},
			want: true,
		},
		{
			name: "dependency variable",
			task: &Task{Deps: []*Dep{{Vars: NewVars(
				&VarElement{Key: "VALUE", Value: Var{Ref: fingerprintRef}},
			)}}},
			want: true,
		},
		{
			name: "precondition",
			task: &Task{Preconditions: []*Precondition{{Msg: fingerprintRef}}},
			want: true,
		},
		{
			name: "unrelated fields and nil entries",
			task: &Task{
				Cmds:          []*Cmd{nil, {Cmd: "echo ok"}},
				Deps:          []*Dep{nil, {Task: "build"}},
				Preconditions: []*Precondition{nil, {Sh: "test -f output"}},
				Status:        []string{"test -f output"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.task.ReferencesFingerprintVar("checksum"))
		})
	}
}
