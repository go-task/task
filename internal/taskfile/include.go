package taskfile

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

var URLReplacer = regexp.MustCompile(`[^a-zA-z0-9]`)

type Includes struct {
	Path          string
	Cache         *time.Duration
	FlushIncludes bool
	Dir           string
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (inc *Includes) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var path string
	if err := unmarshal(&path); err == nil {
		inc.Path = path
		return nil
	}
	var includeStruct struct {
		Path          string
		Cache         *time.Duration
		FlushIncludes bool `yaml:"flush_includes"`
	}
	if err := unmarshal(&includeStruct); err != nil {
		return err
	}
	inc.Path = includeStruct.Path
	inc.Cache = includeStruct.Cache
	inc.FlushIncludes = includeStruct.FlushIncludes
	return nil
}

func (inc *Includes) IsURL() bool {
	if !strings.HasPrefix(inc.Path, "http") {
		return false
	}
	_, err := url.ParseRequestURI(inc.Path)
	return err == nil
}

func (inc *Includes) ensureCacheDir() {
	os.MkdirAll(filepath.Join(inc.Dir, ".task", "includes_cache"), 0755)
}

func (inc *Includes) cacheFilePath() string {
	return filepath.Join(
		inc.Dir,
		".task",
		"includes_cache",
		URLReplacer.ReplaceAllString(inc.Path, "-"),
	)
}

func (inc *Includes) LoadTaskfile() (*Taskfile, error) {
	var taskfile *Taskfile
	var err error
	if inc.IsURL() {
		taskfile, err = inc.loadTaskfileHTTP()
	} else {
		taskfile, err = inc.loadTaskfileLocal()
	}
	if err != nil {
		return nil, err
	}
	if inc.FlushIncludes {
		taskfile.Includes = map[string]*Includes{}
	}
	return taskfile, nil
}

func (inc *Includes) loadTaskfileHTTP() (*Taskfile, error) {
	if !inc.URLCacheExpired() {
		return LoadFromPath(inc.cacheFilePath())
	}
	resp, err := http.Get(inc.Path)
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
			inc.Path, err, body)
	}
	if inc.Cache != nil {
		inc.ensureCacheDir()
		if err = ioutil.WriteFile(inc.cacheFilePath(), body, 0644); err != nil {
			return nil, fmt.Errorf("Unable to cache taskfile %s: %s", inc.Path, err)
		}
	}
	return &t, nil
}

func (inc *Includes) loadTaskfileLocal() (*Taskfile, error) {
	path := filepath.Join(inc.Dir, inc.Path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("Taskfile not found: %s - %s\n", inc.Path, err)
	}
	if info.IsDir() {
		path = filepath.Join(path, "Taskfile.yml")
	}
	return LoadFromPath(path)
}

func (inc *Includes) URLCacheExpired() bool {
	if inc.Cache == nil {
		return true
	}
	fileinfo, err := os.Stat(inc.cacheFilePath())
	if err != nil {
		return true
	}
	return time.Now().After(fileinfo.ModTime().Add(*inc.Cache))
}
