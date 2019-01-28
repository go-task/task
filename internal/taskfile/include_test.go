package taskfile_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/h2non/gock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v2/internal/taskfile"

	"gopkg.in/yaml.v2"
)

var staticIncludedContent = `
version: "2"
vars:
  STATIC_MOO: 1
tasks:
  static_task:
    cmds:
      - echo 1
  dependency_task:
    deps:
      - task: static_task
    cmds:
      - task: static_task
      - echo 1
`

type TestCommandr struct {
	Body     []byte
	Error    error
	Command  string
	Args     []string
	Callback func(string, ...string)
}

func (c TestCommandr) CombinedOutput(command string, args ...string) ([]byte, error) {
	c.Command = command
	c.Args = args
	if c.Callback != nil {
		c.Callback(command, args...)
	}
	return c.Body, c.Error
}

func TestIncludesWithLocalPath(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	content := `
version: '2'
includes:
  moo: ./Taskfile.yml
`
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(tf.Includes))
	afero.WriteFile(taskfile.AppFS, "./Taskfile.yml", []byte(content), 0644)
	err = tf.ProcessIncludes("./")
	assert.EqualError(t, err, taskfile.ErrIncludedTaskfilesCantHaveIncludes.Error())
}

func TestIncludesWithRemotePath(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	defer gock.Off()
	content := `
version: '2'
includes:
  moo:
    path: http://example.com/taskfile.yml
`
	gock.New("http://example.com").Get("/taskfile.yml").Reply(200).BodyString(content)
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.EqualError(t, err, taskfile.ErrIncludedTaskfilesCantHaveIncludes.Error())
}

func TestIncludesWithRemotePathAndCachesReceivedContent(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	defer gock.Off()
	content := `
version: '2'
includes:
  moo:
    cache: 30s
    path: http://example.com/taskfile.yml
`
	gock.New("http://example.com").Get("/taskfile.yml").Reply(200).BodyString(staticIncludedContent)
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	_, err = taskfile.AppFS.Stat(".task/cache/include/http---example-com-taskfile-yml")
	assert.NoError(t, err)
}

func TestIncludesWithRemotePathAndEvictsCache(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	defer gock.Off()
	content := `
version: '2'
includes:
  moo:
    cache: 1ns
    path: http://example.com/taskfile.yml
`
	gock.New("http://example.com").Get("/taskfile.yml").Reply(200).BodyString(staticIncludedContent)
	gock.New("http://example.com").Get("/taskfile.yml").Reply(200).BodyString(staticIncludedContent)
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	_, err = taskfile.AppFS.Stat(".task/cache/include/http---example-com-taskfile-yml")
	assert.NoError(t, err)
}

func TestIncludesWithRemotePathAndCachesReceivedContentUsingDefaults(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	defer gock.Off()
	content := `
version: '2'
includes:
  .defaults:
    cache: 30s
  moo:
    path: http://example.com/taskfile.yml
`
	gock.New("http://example.com").Get("/taskfile.yml").Reply(200).BodyString(staticIncludedContent)
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	assert.Contains(t, tf.Tasks, "moo:static_task")
	assert.Contains(t, tf.Vars, "STATIC_MOO")
	_, err = taskfile.AppFS.Stat(".task/cache/include/http---example-com-taskfile-yml")
	assert.NoError(t, err)
}

func TestIncludesWithRemotePathWithInvalidYaml(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	defer gock.Off()
	content := `
version: '2'
includes:
  moo:
    cache: 1ns
    path: http://example.com/taskfile.yml
`
	gock.New("http://example.com").Get("/taskfile.yml").Reply(404).BodyString(`invalid: "yaml`)
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	err_msg := `Error parsing loading taskfile from http://example.com/taskfile.yml: ` +
		"yaml: found unexpected end of stream - invalid: \"yaml\n"
	assert.EqualError(t, err, err_msg)

}

var gitFileName string

func stubGitClone(taskPath string) func(string, ...string) {
	return func(command string, args ...string) {
		for _, arg := range args {
			if strings.HasPrefix(arg, "/tmp") {
				taskfile.AppFS.MkdirAll(arg, 0777)
				gitFileName = arg
				f, _ := taskfile.AppFS.Create(filepath.Join(arg, taskPath))
				f.WriteString(staticIncludedContent)
				f.Close()
			}
		}
	}
}

func TestIncludesWithGitPathAndCachesReceivedContentUsingDefaults(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	taskfile.ShellCommandr = TestCommandr{
		Body: []byte(staticIncludedContent), Callback: stubGitClone("Taskfile.yml"),
	}
	content := `
version: '2'
includes:
  .defaults:
    cache: 30s
  moo:
    path: git+ssh://example.com:222
`
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	assert.Contains(t, tf.Tasks, "moo:static_task")
	assert.Contains(t, tf.Vars, "STATIC_MOO")
	assert.Equal(t, tf.Tasks["moo:static_task"].Hidden, false)
	_, err = taskfile.AppFS.Stat(gitFileName)
	assert.Error(t, err)
	_, err = taskfile.AppFS.Stat(".task/cache/include/git-ssh---example-com-222")
	assert.NoError(t, err)
}

func TestIncludesWithGitPathAndImportTasksAsHidden(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	taskfile.ShellCommandr = TestCommandr{
		Body: []byte(staticIncludedContent), Callback: stubGitClone("Taskfile.yml"),
	}
	content := `
version: '2'
includes:
  .moo:
    path: git+ssh://example.com:222
    hidden: true
`
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	assert.Contains(t, tf.Tasks, "moo:static_task")
	assert.Equal(t, tf.Tasks["moo:static_task"].Hidden, true)
	_, err = taskfile.AppFS.Stat(gitFileName)
	assert.Error(t, err)
	_, err = taskfile.AppFS.Stat(".task/cache/include/git-ssh---example-com-222")
	assert.Error(t, err)
}

func TestIncludesWithGitPathAndImportTasksAsHiddenFromNamespace(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	taskfile.ShellCommandr = TestCommandr{
		Body: []byte(staticIncludedContent), Callback: stubGitClone("Taskfile.yml"),
	}
	content := `
version: '2'
includes:
  .moo: git+ssh://example.com:222
`
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	assert.Contains(t, tf.Tasks, "moo:static_task")
	assert.Equal(t, tf.Tasks["moo:static_task"].Hidden, true)
	_, err = taskfile.AppFS.Stat(gitFileName)
	assert.Error(t, err)
	_, err = taskfile.AppFS.Stat(".task/cache/include/git-ssh---example-com-222")
	assert.Error(t, err)
}

func TestIncludesWithGitPathAndImportTasksAsDirect(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	taskfile.ShellCommandr = TestCommandr{
		Body: []byte(staticIncludedContent), Callback: stubGitClone("Taskfile.yml"),
	}
	content := `
version: '2'
includes:
  moo:
    path: git+ssh://example.com:222
    direct: true
`
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	assert.Contains(t, tf.Tasks, "moo:static_task")
	assert.Equal(t, tf.Tasks["moo:static_task"].Hidden, false)
	assert.Equal(t, tf.Tasks["static_task"].Hidden, false)
	_, err = taskfile.AppFS.Stat(gitFileName)
	assert.Error(t, err)
	_, err = taskfile.AppFS.Stat(".task/cache/include/git-ssh---example-com-222")
	assert.Error(t, err)
}

func TestIncludesWithGitPathAndImportTasksAsDirectWithoutOverride(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	taskfile.ShellCommandr = TestCommandr{
		Body: []byte(staticIncludedContent), Callback: stubGitClone("Taskfile.yml"),
	}
	content := `
version: '2'
includes:
  _moo: git+ssh://example.com:222

tasks:
  static_task:
    desc: "moo"
`
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	assert.Contains(t, tf.Tasks, "moo:static_task")
	assert.Equal(t, tf.Tasks["moo:static_task"].Hidden, true)
	assert.Equal(t, tf.Tasks["static_task"].Hidden, false)
	assert.Equal(t, tf.Tasks["static_task"].Desc, "moo")
	_, err = taskfile.AppFS.Stat(gitFileName)
	assert.Error(t, err)
	_, err = taskfile.AppFS.Stat(".task/cache/include/git-ssh---example-com-222")
	assert.Error(t, err)
}

func TestIncludesWithGitPathAndCheckDependencyInheritance(t *testing.T) {
	taskfile.AppFS = afero.NewMemMapFs()
	taskfile.ShellCommandr = TestCommandr{
		Body: []byte(staticIncludedContent), Callback: stubGitClone("Taskfile.yml"),
	}
	content := `
version: '2'
includes:
  _moo: git+ssh://example.com:222
`
	var tf *taskfile.Taskfile
	err := yaml.Unmarshal([]byte(content), &tf)
	assert.NoError(t, err)
	err = tf.ProcessIncludes("./")
	assert.NoError(t, err)
	assert.Contains(t, tf.Tasks, "moo:static_task")
	assert.Contains(t, tf.Tasks, "moo:dependency_task")

	cmds := []string{}
	for _, cmd := range tf.Tasks["moo:dependency_task"].Cmds {
		cmds = append(cmds, cmd.Task)
	}
	assert.Contains(t, cmds, "moo:static_task")

	cmds = []string{}
	for _, cmd := range tf.Tasks["dependency_task"].Cmds {
		cmds = append(cmds, cmd.Task)
	}
	assert.Contains(t, cmds, "moo:static_task")

	cmds = []string{}
	for _, cmd := range tf.Tasks["moo:dependency_task"].Deps {
		cmds = append(cmds, cmd.Task)
	}
	assert.Contains(t, cmds, "moo:static_task")

	cmds = []string{}
	for _, cmd := range tf.Tasks["dependency_task"].Deps {
		cmds = append(cmds, cmd.Task)
	}
	assert.Contains(t, cmds, "moo:static_task")
}
