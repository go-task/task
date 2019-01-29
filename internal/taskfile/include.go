package taskfile

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

var (
	// URLReplacer is a regexp for generating safe filenames form URLs
	URLReplacer   = regexp.MustCompile(`[^a-zA-z0-9]`)
	ShellCommandr ShellOutCommander
)

func init() {
	ShellCommandr = commandr{}
}

type ShellOutCommander interface {
	CombinedOutput(string, ...string) ([]byte, error)
}

type commandr struct{}

func (c commandr) CombinedOutput(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).CombinedOutput()
}

type Includes []*Include

type Include struct {
	Namespace string

	// Path should be a local path or a remote URL
	Path string

	// Cache specifies a duration how long the remote Taskfile should be cached.
	// Syntax should be according the `time.ParseDuration` function:
	// https://golang.org/pkg/time/#ParseDuration
	Cache *time.Duration `yaml:"cache"`

	// Dir holds the working directory.
	Dir string

	// Hidden flag will import tasks with hidden flags, alternatively prefix
	// your task with a dot. Example: ".ruby"
	Hidden bool `yaml:"hidden"`

	// Direct flag will import tasks without applying namespaces, alternatively prefix
	// your task with a underscore. Example: "_ruby"
	// The tasks will be accessiable via the namespace as well and when using the
	// mentioned short version above, it will be marked as hidden as well.
	Direct bool `yaml:"direct"`
}

// IncludesFromYaml parses the yaml manually to perserve the include order.
// It will return the include objects as first param and the defaults as second one.
func IncludesFromYaml(slice yaml.MapSlice) (Includes, *Include, error) {
	includes := make(Includes, 0)
	var defaultInclude *Include
	for _, mapItem := range slice {
		inc := &Include{}
		switch val := mapItem.Value.(type) {
		case string:
			inc.Path = val
		case yaml.MapSlice:
			for _, innerItem := range val {
				switch innerItem.Key.(string) {
				case "path":
					inc.Path = innerItem.Value.(string)
				case "cache":
					cache := innerItem.Value.(string)
					duration, err := time.ParseDuration(cache)
					if err != nil {
						return nil, nil, fmt.Errorf("task: Unable to parse cache from include. %s", err)
					}
					inc.Cache = &duration
				case "hidden":
					inc.Hidden = innerItem.Value.(bool)
				case "direct":
					inc.Direct = innerItem.Value.(bool)
				}
			}
		}
		inc.Namespace = mapItem.Key.(string)
		if inc.Namespace == ".defaults" {
			defaultInclude = inc
		} else {
			includes = append(includes, inc)
		}
	}
	return includes, defaultInclude, nil

}

func (i *Include) ApplySettingsByNamespace(namespace string) {
	if strings.HasPrefix(namespace, ".") {
		i.Hidden = true
	}
	if strings.HasPrefix(namespace, "_") {
		i.Hidden = true
		i.Direct = true
	}
}

func (i *Include) ApplyDefaults(def *Include) {
	if def.Cache != nil && i.Cache == nil {
		i.Cache = def.Cache
	}
}

// IsURL returns true if the given path seems to be a url
func (i *Include) IsURL() bool {
	if !strings.HasPrefix(i.Path, "http") {
		return false
	}
	_, err := url.ParseRequestURI(i.Path)
	return err == nil
}

// IsGitSSHURL return true if it looks like a ssh+git:// url
func (i *Include) IsGitSSHURL() bool {
	return strings.HasPrefix(i.Path, "git+ssh://")
}

// LoadTaskfile resolves the referenced include and returns a taskfile
func (i *Include) LoadTaskfile() (*Taskfile, error) {
	var taskfile *Taskfile
	var err error
	if i.IsURL() {
		taskfile, err = i.loadTaskfileHTTP()
	} else if i.IsGitSSHURL() {
		taskfile, err = i.loadTaskfileGitSSH()
	} else {
		taskfile, err = i.loadTaskfileLocal()
	}
	if err != nil {
		return nil, err
	}
	return taskfile, nil
}

// URLCacheExpired returns true if the cache isn't configured or the cached file is out of date
func (i *Include) URLCacheExpired() bool {
	if i.Cache == nil {
		return true
	}
	fileinfo, err := AppFS.Stat(i.cacheFilePath())
	if err != nil {
		return true
	}
	return time.Now().After(fileinfo.ModTime().Add(*i.Cache))
}

func (i *Include) cache(body []byte) error {
	if i.Cache != nil {
		if err := i.ensureCacheDir(); err != nil {
			return fmt.Errorf("task: Unable create cache dir: %s", err)
		}

		if err := afero.WriteFile(AppFS, i.cacheFilePath(), body, 0644); err != nil {
			return fmt.Errorf("task: Unable to cache Taskfile %s: %s", i.Path, err)
		}
	}
	return nil
}

func (i *Include) loadTaskfileGitSSH() (*Taskfile, error) {
	if !i.URLCacheExpired() {
		return LoadFromPath(i.cacheFilePath())
	}
	uri, err := url.ParseRequestURI(i.Path)
	user := ""
	if uri.User.Username() != "" {
		user = uri.User.Username() + "@"
	}
	paths := strings.Split(uri.Path, ":")
	path := "Taskfile.yml"
	if len(paths) > 1 {
		path = paths[1]
	}
	cloneDir, err := afero.TempDir(AppFS, "/tmp", "task-git-clone")
	if err != nil {
		return nil, fmt.Errorf("task: Unable to create tmp dir for %s: %s", i.Path, err)
	}
	defer AppFS.RemoveAll(cloneDir)
	out, err := ShellCommandr.CombinedOutput(
		"git", "clone", "--depth", "1",
		fmt.Sprintf("ssh://%s%s%s", user, uri.Host, paths[0]), cloneDir,
	)
	if err != nil {
		return nil, fmt.Errorf("task: Unable to clone %s: %s %s", i.Path, err, out)
	}
	file, err := AppFS.Open(filepath.Join(cloneDir, path))
	if err != nil {
		return nil, fmt.Errorf("task: Unable to open file %s for include %s: %s",
			path, i.Path, err)
	}
	body := new(bytes.Buffer)
	if _, err := io.Copy(body, file); err != nil {
		return nil, fmt.Errorf("task: Unable to buffer file %s for include %s: %s",
			path, i.Path, err)
	}
	var t Taskfile
	if err := yaml.Unmarshal(body.Bytes(), &t); err != nil {
		return nil, fmt.Errorf("Error parsing loading taskfile from %s: %s - %s\n",
			i.Path, err, body)
	}
	return &t, i.cache(body.Bytes())
}

func (i *Include) loadTaskfileHTTP() (*Taskfile, error) {
	if !i.URLCacheExpired() {
		return LoadFromPath(i.cacheFilePath())
	}
	resp, err := http.Get(i.Path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var t Taskfile
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("Error parsing loading taskfile from %s: %s - %s\n",
			i.Path, err, body)
	}
	return &t, i.cache(body)
}

func (i *Include) ensureCacheDir() error {
	return AppFS.MkdirAll(filepath.Join(i.Dir, ".task", "cache", "include"), 0755)
}

func (i *Include) loadTaskfileLocal() (*Taskfile, error) {
	path := filepath.Join(i.Dir, i.Path)
	info, err := AppFS.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("task: Taskfile not found: %s - %s\n", i.Path, err)
	}
	if info.IsDir() {
		path = filepath.Join(path, "Taskfile.yml")
	}
	return LoadFromPath(path)
}

func (i *Include) cacheFilePath() string {
	return filepath.Join(
		i.Dir,
		".task",
		"cache",
		"include",
		URLReplacer.ReplaceAllString(i.Path, "-"),
	)
}
