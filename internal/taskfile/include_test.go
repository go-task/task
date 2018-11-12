package taskfile_test

import (
	"testing"

	"github.com/h2non/gock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v2/internal/taskfile"

	"gopkg.in/yaml.v2"
)

var staticIncludedContent = `version: "2"`

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
