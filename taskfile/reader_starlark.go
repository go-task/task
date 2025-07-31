package taskfile

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/go-task/task/v3/taskfile/ast"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func defTaskImpl(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var taskName string
	var cmds starlark.Value
	var deps starlark.Value
	var label string
	var desc string
	var sources starlark.Value
	var status starlark.Value
	var silent bool

	if err := starlark.UnpackArgs(
		fn.Name(), args, kwargs,
		"name", &taskName,
		"cmds?", &cmds,
		"deps?", &deps,
		"label?", &label,
		"desc?", &desc,
		"sources?", &sources,
		"status?", &status,
		"silent?", &silent); err != nil {
		return nil, err
	}

	frame := thread.CallFrame(0)
	task := &ast.Task{
		Task:  taskName,
		Label: label,
		Desc:  desc,
		Location: &ast.Location{
			Line:     int(frame.Pos.Line),
			Column:   int(frame.Pos.Col),
			Taskfile: frame.Pos.Filename(),
		},
	}

	tasks := thread.Local("tasks").(*ast.Tasks)
	tasks.Set(taskName, task)
	return starlark.None, nil
}

func readStarlarkTaskfile(entrypoint string, src []byte, tf *ast.Taskfile) error {
	var err error

	thread := &starlark.Thread{}
	thread.SetLocal("tasks", ast.NewTasks())

	fileOpts := &syntax.FileOptions{}
	predeclared := starlark.StringDict{
		"def_task": starlark.NewBuiltin("def_task", defTaskImpl),
	}

	globals, err := starlark.ExecFileOptions(fileOpts, thread, entrypoint, src, predeclared)
	if err != nil {
		return err
	}

	if globals.Has("version") {
		version, didParse := starlark.AsString(globals["version"])
		if !didParse {
			return fmt.Errorf("expected version to be string, got %s", globals["version"].Type())
		}

		tf.Version = semver.MustParse(version)
	}

	tf.Tasks = thread.Local("tasks").(*ast.Tasks)

	return nil
}
