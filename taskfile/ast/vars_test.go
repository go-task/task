package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVars_ToCacheMap(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns nil", func(t *testing.T) {
		t.Parallel()
		var vars *Vars
		assert.Nil(t, vars.ToCacheMap())
	})

	t.Run("empty vars returns empty map", func(t *testing.T) {
		t.Parallel()
		vars := NewVars()
		m := vars.ToCacheMap()
		assert.NotNil(t, m)
		assert.Empty(t, m)
	})

	t.Run("static values are included", func(t *testing.T) {
		t.Parallel()
		vars := NewVars(
			&VarElement{Key: "FOO", Value: Var{Value: "bar"}},
			&VarElement{Key: "NUM", Value: Var{Value: 42}},
		)
		m := vars.ToCacheMap()
		assert.Equal(t, map[string]any{"FOO": "bar", "NUM": 42}, m)
	})

	t.Run("live values take precedence over static values", func(t *testing.T) {
		t.Parallel()
		vars := NewVars(
			&VarElement{Key: "FOO", Value: Var{Value: "bar", Live: "live-bar"}},
		)
		m := vars.ToCacheMap()
		assert.Equal(t, map[string]any{"FOO": "live-bar"}, m)
	})

	t.Run("dynamic variables are excluded", func(t *testing.T) {
		t.Parallel()
		sh := "echo hello"
		vars := NewVars(
			&VarElement{Key: "STATIC", Value: Var{Value: "ok"}},
			&VarElement{Key: "DYNAMIC", Value: Var{Sh: &sh}},
		)
		m := vars.ToCacheMap()
		assert.Equal(t, map[string]any{"STATIC": "ok"}, m)
	})
}
