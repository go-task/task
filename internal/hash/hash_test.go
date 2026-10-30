package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

// resolveTo resolves each name to the id given for it, standing in for the
// executor's lookup of what a reference runs.
func resolveTo(ids map[string]string) Resolve {
	return func(name string) (string, bool) {
		id, ok := ids[name]
		return id, ok
	}
}

var (
	loc      = &ast.Location{Taskfile: "/lib/Taskfile.yml", Line: 4, Column: 3}
	resolver = resolveTo(map[string]string{
		// One definition reached through two include paths.
		"group:service-a:library:generate": "generate",
		"group:service-b:library:generate": "generate",
		// A reference escaping to each path's own includer.
		"group:service-a:setup": "setup-a",
		"group:service-b:setup": "setup-b",
	})
)

func task(fullName, dep, cmd string, match ...string) *ast.Task {
	vars := ast.NewVars()
	if match != nil {
		vars.Set("MATCH", ast.Var{Value: match})
	}
	return &ast.Task{
		FullName: fullName,
		Location: loc,
		Vars:     vars,
		Deps:     []*ast.Dep{{Task: dep}},
		Cmds:     []*ast.Cmd{{Cmd: cmd}},
	}
}

func TestName(t *testing.T) {
	t.Parallel()

	name := func(tk *ast.Task) string {
		n, err := Name(tk, resolver)
		require.NoError(t, err)
		return n
	}

	// Deps that resolve to one definition: the calls run the same tasks.
	assert.Equal(t,
		name(task("group:service-a:library:build", "group:service-a:library:generate", "echo lib")),
		name(task("group:service-b:library:build", "group:service-b:library:generate", "echo lib")),
	)
	// Deps that resolve to two definitions: each call runs its own.
	assert.NotEqual(t,
		name(task("group:service-a:library:build", "group:service-a:setup", "echo lib")),
		name(task("group:service-b:library:build", "group:service-b:setup", "echo lib")),
	)
	// One wildcard definition called under two matches.
	assert.NotEqual(t,
		name(task("deploy:infra", "", "deploy", "infra")),
		name(task("deploy:js", "", "deploy", "js")),
	)
	// Variables do not count: that is run: when_changed.
	assert.Equal(t,
		name(task("library:build", "group:service-a:library:generate", "echo foo")),
		name(task("library:build", "group:service-a:library:generate", "echo bar")),
	)
}

func TestHash(t *testing.T) {
	t.Parallel()

	hash := func(tk *ast.Task) string {
		h, err := Hash(tk, resolver)
		require.NoError(t, err)
		return h
	}

	assert.Equal(t,
		hash(task("group:service-a:library:build", "group:service-a:library:generate", "echo lib")),
		hash(task("group:service-b:library:build", "group:service-b:library:generate", "echo lib")),
	)
	assert.NotEqual(t,
		hash(task("group:service-a:library:build", "group:service-a:setup", "echo lib")),
		hash(task("group:service-b:library:build", "group:service-b:setup", "echo lib")),
	)
	// Different call variables compile to different commands.
	assert.NotEqual(t,
		hash(task("library:build", "group:service-a:library:generate", "echo foo")),
		hash(task("library:build", "group:service-a:library:generate", "echo bar")),
	)
}
