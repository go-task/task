package exporter

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"os"
	"testing"
)

func TestGithubActionsExporter_Export(t *testing.T) {
	tmpUuid := uuid.New()
	var tmpUuidGenerator uuidGenerator = func() uuid.UUID { return tmpUuid }
	tmpFileName := func(subtest string) string {
		return fmt.Sprintf("%s/%s_%s", t.TempDir(), subtest, "github_actions_env.test")
	}

	type fields struct {
		envFilePath   string
		uuidGenerator uuidGenerator
		eol           string
	}
	type args struct {
		vars map[string]string
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantFileContent string
		wantErr         bool
	}{
		{
			name:            "validSingleKeys",
			fields:          fields{envFilePath: tmpFileName("0"), uuidGenerator: tmpUuidGenerator, eol: "\n"},
			args:            args{vars: map[string]string{"key": "value"}},
			wantFileContent: fmt.Sprintf("key<<ghadelimiter_%s\nvalue\nghadelimiter_%s\n", tmpUuid, tmpUuid),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &GithubActionsExporter{
				envFilePath:   tt.fields.envFilePath,
				uuidGenerator: tt.fields.uuidGenerator,
				eol:           tt.fields.eol,
			}
			if err := e.Export(tt.args.vars); (err != nil) != tt.wantErr {
				t.Errorf("Export() error = %v, wantErr %v", err, tt.wantErr)
			}
			gotContents, err := readFile(tt.fields.envFilePath)
			if err != nil {
				t.Errorf("Export() content reading error: %v", err)
			}
			if gotContents != tt.wantFileContent {
				t.Errorf("Export() content = %s, want: %s", gotContents, tt.wantFileContent)
			}
		})
	}
}

func TestGithubActionsExporter_prepareKeyValueMessage(t *testing.T) {
	tmpUuid := uuid.New()
	var tmpUuidGenerator uuidGenerator = func() uuid.UUID { return tmpUuid }

	type fields struct {
		envFilePath   string
		uuidGenerator uuidGenerator
		eol           string
	}
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name:   "validSingleLine",
			fields: fields{uuidGenerator: tmpUuidGenerator, eol: "\n"},
			args:   args{"test_key", "test_value"},
			want:   fmt.Sprintf("test_key<<ghadelimiter_%s\ntest_value\nghadelimiter_%s\n", tmpUuid, tmpUuid),
		},
		{
			name:   "validMultiLine",
			fields: fields{uuidGenerator: tmpUuidGenerator, eol: "\n"},
			args:   args{"test_key", "test_value\ntestanotherline"},
			want:   fmt.Sprintf("test_key<<ghadelimiter_%s\ntest_value\ntestanotherline\nghadelimiter_%s\n", tmpUuid, tmpUuid),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &GithubActionsExporter{
				envFilePath:   tt.fields.envFilePath,
				uuidGenerator: tt.fields.uuidGenerator,
				eol:           tt.fields.eol,
			}
			if got := e.prepareKeyValueMessage(tt.args.key, tt.args.value); got != tt.want {
				t.Errorf("prepareKeyValueMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGithubActionsExporter(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		env     string
		wantErr bool
	}{
		{name: "validGithubEnv", env: "/foo/bar/github.env", want: "/foo/bar/github.env"},
		{name: "invalidGithubEnv", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env == "" {
				_ = os.Unsetenv(gitHubEnvName)
			} else {
				t.Setenv(gitHubEnvName, tt.env)
			}

			got, err := NewGithubActionsExporter()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGithubActionsExporter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != nil && got.envFilePath != tt.want {
				t.Errorf("NewGithubActionsExporter() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func readFile(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
