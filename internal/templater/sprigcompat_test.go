package templater

import (
	"testing"

	"github.com/go-task/task/v3/taskfile/ast"
)

// render evaluates a single template expression and returns its output, or the
// error the template engine raised.
func render(t *testing.T, expr string) (string, error) {
	t.Helper()
	cache := &Cache{Vars: ast.NewVars()}
	got := ReplaceWithExtra(expr, cache, nil)
	if err := cache.Err(); err != nil {
		return "", err
	}
	return got, nil
}

// The ten functions whose argument order changed between slim-sprig and sprout
// must accept both, so that existing Taskfiles keep working while new ones can
// use the pipe form.
func TestSprigSignatureShims(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		sprig  string
		sprout string
		want   string
	}{
		{"get", `{{ get (dict "a" "b") "a" }}`, `{{ dict "a" "b" | get "a" }}`, "b"},
		{"hasKey", `{{ hasKey (dict "a" "b") "a" }}`, `{{ dict "a" "b" | hasKey "a" }}`, "true"},
		{"unset", `{{ unset (dict "a" "b" "c" "d") "a" | toJson }}`, `{{ dict "a" "b" "c" "d" | unset "a" | toJson }}`, `{"c":"d"}`},
		{"set", `{{ set (dict "a" "b") "c" "d" | toJson }}`, `{{ dict "a" "b" | set "c" "d" | toJson }}`, `{"a":"b","c":"d"}`},
		{"pick", `{{ pick (dict "a" "1" "b" "2") "a" | toJson }}`, `{{ dict "a" "1" "b" "2" | pick "a" | toJson }}`, `{"a":"1"}`},
		{"omit", `{{ omit (dict "a" "1" "b" "2") "a" | toJson }}`, `{{ dict "a" "1" "b" "2" | omit "a" | toJson }}`, `{"b":"2"}`},
		{"append", `{{ append (list 1 2) 3 | toJson }}`, `{{ list 1 2 | append 3 | toJson }}`, "[1,2,3]"},
		{"prepend", `{{ prepend (list 2 3) 1 | toJson }}`, `{{ list 2 3 | prepend 1 | toJson }}`, "[1,2,3]"},
		{"without", `{{ without (list 1 2 3) 2 | toJson }}`, `{{ list 1 2 3 | without 2 | toJson }}`, "[1,3]"},
		{"slice", `{{ slice (list 1 2 3 4) 1 3 | toJson }}`, `{{ list 1 2 3 4 | slice 1 3 | toJson }}`, "[2,3]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for order, expr := range map[string]string{"slim-sprig": tt.sprig, "sprout": tt.sprout} {
				got, err := render(t, expr)
				if err != nil {
					t.Fatalf("%s order: %s unexpected error: %v", order, expr, err)
				}
				if got != tt.want {
					t.Errorf("%s order: %s = %q; want %q", order, expr, got, tt.want)
				}
			}
		})
	}
}

// sprout dropped sprig's default-value argument on dig. Task keeps it, since
// the two forms are indistinguishable by type and every existing Taskfile was
// written against sprig's.
func TestDigKeepsSprigDefault(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		expr string
		want string
	}{
		{"path found", `{{ dig "a" "b" "fallback" (dict "a" (dict "b" "found")) }}`, "found"},
		{"key missing", `{{ dig "a" "missing" "fallback" (dict "a" (dict "b" "found")) }}`, "fallback"},
		{"root missing", `{{ dig "nope" "fallback" (dict "a" 1) }}`, "fallback"},
		{"leaf is not a dict", `{{ dig "a" "b" "c" "fallback" (dict "a" (dict "b" "found")) }}`, "fallback"},
		// sprout splits keys on dots, which sprig did not. Kept: it is an
		// improvement and no slim-sprig key could contain a dot anyway.
		{"dotted path", `{{ dig "a.b" "fallback" (dict "a" (dict "b" "found")) }}`, "found"},
		// Two arguments is unambiguously sprout's no-default form.
		{"no default", `{{ dig "a" (dict "a" "found") }}`, "found"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := render(t, tt.expr)
			if err != nil {
				t.Fatalf("%s unexpected error: %v", tt.expr, err)
			}
			if got != tt.want {
				t.Errorf("%s = %q; want %q", tt.expr, got, tt.want)
			}
		})
	}
}

// Task's own functions must keep winning over the sprout functions and aliases
// that share their name, since their semantics differ.
func TestTaskFuncsShadowSprout(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		expr string
		want string
	}{
		// sprout's merge is a deep merge that keeps the destination value;
		// Task's is a shallow merge where the last map wins.
		{"merge overwrites", `{{ merge (dict "a" 1) (dict "a" 0) | toJson }}`, `{"a":0}`},
		// sprout aliases fromYaml/toYaml onto fromYAML/toYAML, which raise
		// errors where Task's swallow them.
		{"fromYaml swallows errors", `{{ fromYaml "a: :" | toJson }}`, "null"},
		{"toYaml", `{{ toYaml (dict "a" 1) }}`, "a: 1\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := render(t, tt.expr)
			if err != nil {
				t.Fatalf("%s unexpected error: %v", tt.expr, err)
			}
			if got != tt.want {
				t.Errorf("%s = %q; want %q", tt.expr, got, tt.want)
			}
		})
	}
}
