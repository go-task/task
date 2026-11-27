package execext

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCoreutilsMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value    string
		expected CoreutilsMode
		wantErr  bool
	}{
		{value: "", expected: CoreutilsAuto},
		{value: "turtle", expected: CoreutilsTurtle},
		{value: "system", expected: CoreutilsSystem},
		{value: "auto", expected: CoreutilsAuto},
		{value: "off", expected: CoreutilsOff},
		{value: "bogus", wantErr: true},
		{value: "true", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			mode, err := ParseCoreutilsMode(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, mode)
		})
	}
}

func TestCoreutilsFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		alias     string
		legacy    string
		expected  CoreutilsMode
		expectSet bool
	}{
		{name: "unset", expected: CoreutilsAuto},
		{name: "alias mode", alias: "turtle", expected: CoreutilsTurtle, expectSet: true},
		{name: "alias bool true", alias: "1", expected: CoreutilsSystem, expectSet: true},
		{name: "alias bool false", alias: "0", expected: CoreutilsAuto, expectSet: true},
		{name: "legacy mode", legacy: "system", expected: CoreutilsSystem, expectSet: true},
		{name: "legacy bool true", legacy: "true", expected: CoreutilsSystem, expectSet: true},
		{name: "legacy bool false", legacy: "false", expected: CoreutilsAuto, expectSet: true},
		{name: "alias wins", alias: "off", legacy: "turtle", expected: CoreutilsOff, expectSet: true},
		{name: "invalid alias falls back to legacy", alias: "bogus", legacy: "turtle", expected: CoreutilsTurtle, expectSet: true},
		{name: "invalid values are ignored", alias: "bogus", legacy: "nope", expected: CoreutilsAuto},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TASK_COREUTILS", tt.alias)
			t.Setenv("TASK_CORE_UTILS", tt.legacy)

			mode, set := CoreutilsFromEnv()
			assert.Equal(t, tt.expected, mode)
			assert.Equal(t, tt.expectSet, set)
		})
	}
}

func TestResolveCoreutils(t *testing.T) {
	tests := []struct {
		name     string
		taskrc   string
		alias    string
		legacy   string
		expected CoreutilsMode
		wantErr  bool
	}{
		{name: "default", taskrc: "", alias: "", legacy: "", expected: CoreutilsAuto},
		{name: "env wins over taskrc", taskrc: "turtle", alias: "off", expected: CoreutilsOff},
		{name: "taskrc invalid", taskrc: "bogus", wantErr: true},
		{name: "taskrc fallback", taskrc: "turtle", expected: CoreutilsTurtle},
		{name: "legacy fallback", legacy: "off", expected: CoreutilsOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TASK_COREUTILS", tt.alias)
			t.Setenv("TASK_CORE_UTILS", tt.legacy)

			mode, err := ResolveCoreutils(tt.taskrc)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, mode)
		})
	}
}

func TestExecHandlers(t *testing.T) {
	t.Parallel()

	assert.Empty(t, execHandlers(CoreutilsOff))
	assert.Len(t, execHandlers(CoreutilsTurtle), 1)
	assert.Len(t, execHandlers(CoreutilsSystem), 1)
	assert.Len(t, execHandlers(CoreutilsAuto), 1)
}
