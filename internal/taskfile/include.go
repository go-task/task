package taskfile

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

var (
	// URLReplacer is a regexp for generating safe filenames form URLs
	URLReplacer = regexp.MustCompile(`[^a-zA-z0-9]`)
)

type Include struct {
	// Path should be a local path or a remote URL
	Path string

	// Cache specifies a duration how long the remote Taskfile should be cached.
	// Syntax should be according the `time.ParseDuration` function:
	// https://golang.org/pkg/time/#ParseDuration
	Cache *time.Duration `yaml:"cache"`

	// Dir holds the working directory.
	Dir string
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (i *Include) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var path string
	if err := unmarshal(&path); err == nil {
		i.Path = path
		return nil
	}
	var includeStruct struct {
		Path  string
		Cache *time.Duration
	}
	if err := unmarshal(&includeStruct); err != nil {
		return err
	}
	i.Path = includeStruct.Path
	i.Cache = includeStruct.Cache
	return nil
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

// LoadTaskfile resolves the referenced include and returns a taskfile
func (i *Include) LoadTaskfile() (*Taskfile, error) {
	var taskfile *Taskfile
	var err error
	if i.IsURL() {
		taskfile, err = i.loadTaskfileHTTP()
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
	if i.Cache != nil {
		if err := i.ensureCacheDir(); err != nil {
			return nil, fmt.Errorf("task: Unable create cache dir: %s", err)
		}

		if err = afero.WriteFile(AppFS, i.cacheFilePath(), body, 0644); err != nil {
			return nil, fmt.Errorf("task: Unable to cache Taskfile %s: %s", i.Path, err)
		}
	}
	return &t, nil
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
