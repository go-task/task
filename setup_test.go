package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestDetectCIOutput(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "no CI detected", env: nil, want: ""},
		{name: "GITLAB_CI=true", env: map[string]string{"GITLAB_CI": "true"}, want: "gitlab"},
		{name: "GITLAB_CI=1", env: map[string]string{"GITLAB_CI": "1"}, want: "gitlab"},
		{name: "GITLAB_CI=false", env: map[string]string{"GITLAB_CI": "false"}, want: ""},
		{name: "GITLAB_CI empty", env: map[string]string{"GITLAB_CI": ""}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GITLAB_CI", "") // reset
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			assert.Equal(t, tc.want, detectCIOutput())
		})
	}
}

func TestSetupOutputPriority(t *testing.T) {
	cases := []struct {
		name          string
		cliStyle      ast.Output
		taskfileStyle ast.Output
		ciAuto        bool
		gitlabEnv     string
		wantName      string
	}{
		{
			name:     "CLI wins over everything",
			cliStyle: ast.Output{Name: "prefixed"},
			taskfileStyle: ast.Output{Name: "group", Group: ast.OutputGroup{
				Begin: "b", End: "e",
			}},
			ciAuto:    true,
			gitlabEnv: "true",
			wantName:  "prefixed",
		},
		{
			name:          "Taskfile wins over auto-detect",
			taskfileStyle: ast.Output{Name: "prefixed"},
			ciAuto:        true,
			gitlabEnv:     "true",
			wantName:      "prefixed",
		},
		{
			name:      "auto-detect activates when nothing explicit",
			ciAuto:    true,
			gitlabEnv: "true",
			wantName:  "gitlab",
		},
		{
			name:      "auto-detect disabled does nothing",
			ciAuto:    false,
			gitlabEnv: "true",
			wantName:  "",
		},
		{
			name:      "auto-detect without CI env does nothing",
			ciAuto:    true,
			gitlabEnv: "",
			wantName:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GITLAB_CI", tc.gitlabEnv)

			e := &Executor{
				OutputStyle:  tc.cliStyle,
				OutputCIAuto: tc.ciAuto,
				Taskfile:     &ast.Taskfile{Output: tc.taskfileStyle},
				Logger:       &logger.Logger{},
			}
			require.NoError(t, e.setupOutput())
			assert.Equal(t, tc.wantName, e.OutputStyle.Name)
		})
	}
}
