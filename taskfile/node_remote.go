package taskfile

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/hashicorp/go-getter"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
)

// An RemoteNode is a node that reads a Taskfile from a remote location using go-getter's URL syntax
//
// See https://pkg.go.dev/github.com/hashicorp/go-getter#readme-url-format for details.
type RemoteNode struct {
	*BaseNode

	cachedSource     *source
	client           *getter.Client
	logger           *logger.Logger
	proto            string
	taskfileOverride string
	timeout          time.Duration
	url              *url.URL
}

func NewRemoteNode(
	l *logger.Logger,
	entrypoint string,
	dir string,
	insecure bool,
	timeout time.Duration,
	opts ...NodeOption,
) (*RemoteNode, bool, error) {

	client := newGetterClient(dir)
	proto, u, err := extractProtocolFromURL(client, entrypoint)
	if err != nil {
		return nil, false, fmt.Errorf("error parsing remote protocol %s: %w", entrypoint, err)
	}

	if proto == "file" {
		// We don't support go-getter's file implementation because it doesn't give direct access to the file,
		// returning a symlink instead.  We prefer to rely on our own implementation.
		return nil, false, nil
	}

	u, err = resolveHTTPEntrypoint(l, insecure, timeout, u)
	if err != nil {
		return nil, false, err
	}

	var tf string
	tf, u = resolveTaskfileOverride(u)

	return &RemoteNode{
		BaseNode:         NewBaseNode(dir, opts...),
		client:           client,
		logger:           l,
		proto:            proto,
		taskfileOverride: tf,
		timeout:          timeout,
		url:              u,
	}, true, nil
}

func (r *RemoteNode) Location() string {
	return r.proto + "::" + r.url.String()
}

func (r *RemoteNode) Remote() bool {
	return true
}

func (r *RemoteNode) Read(ctx context.Context) (*source, error) {
	return r.loadSource(ctx)
}

func (r *RemoteNode) ResolveEntrypoint(entrypoint string) (string, error) {
	childProto, _, err := extractProtocolFromURL(r.client, entrypoint)
	if err != nil {
		return "", fmt.Errorf("could not resolve protocol for include %s: %w", entrypoint, err)
	}

	switch {
	case childProto != "file":
		return entrypoint, nil

	case filepath.IsAbs(entrypoint):
		return entrypoint, nil

	case r.proto == "http" || r.proto == "https":
		// In HTTP, relative includes aren't available locally and are downloaded from the same base URL.
		base := *r.url
		base.Path = filepath.Join(filepath.Dir(base.Path), entrypoint)

		return base.String(), nil

	default:
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		src, err := r.loadSource(ctx)
		if err != nil {
			return "", err
		}

		return filepathext.SmartJoin(src.FileDirectory, entrypoint), nil
	}
}

func (r *RemoteNode) ResolveDir(dir string) (string, error) {
	path, err := execext.Expand(dir)
	if err != nil {
		return "", err
	}

	if filepathext.IsAbs(path) {
		return path, nil
	}

	// NOTE: Uses the directory of the entrypoint (Taskfile), not the current working directory
	// This means that files are included relative to one another
	entrypointDir := filepath.Dir(r.Dir())
	return filepathext.SmartJoin(entrypointDir, path), nil
}

func (r *RemoteNode) FilenameAndLastDir() (string, string) {
	u, path := getter.SourceDirSubdir(r.url.String())

	u2, _ := url.Parse(u)
	fullPath := filepath.Join(u2.Path, path)

	dir, filename := filepath.Split(fullPath)
	return filepath.Base(dir), filename
}

func (r *RemoteNode) loadSource(ctx context.Context) (*source, error) {
	if r.cachedSource == nil {
		r.logger.VerboseOutf(logger.Magenta, "task: [%s] Fetching remote taskfile from %s\n", r.Location(), r.client.Src)

		dir, err := os.MkdirTemp("", "taskfile-remote-")
		if err != nil {
			return nil, err
		}
		r.client.Ctx = ctx
		r.client.Src = r.Location()
		r.client.Dst = dir

		if err := r.client.Get(); err != nil {
			return nil, err
		}

		taskfile, err := r.resolveTaskfilePath(r.logger, dir)
		if err != nil {
			return nil, err
		}

		b, err := os.ReadFile(taskfile)
		if err != nil {
			return nil, err
		}

		r.cachedSource = &source{
			FileContent:   b,
			FileDirectory: filepath.Dir(taskfile),
			Filename:      filepath.Base(taskfile),
		}
	}

	return r.cachedSource, nil
}

func (r *RemoteNode) resolveTaskfilePath(l *logger.Logger, dir string) (string, error) {
	// If there's a single file in the directory, use that
	// If there's a default taskfile name, use that
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	if len(entries) == 1 {
		return filepath.Join(dir, entries[0].Name()), nil
	}

	if r.taskfileOverride != "" {
		return filepath.Join(dir, r.taskfileOverride), nil
	}

	return Exists(l, dir)
}

func newGetterClient(dir string) *getter.Client {
	httpGetter := &httpGetter{
		HttpGetter: &getter.HttpGetter{
			XTerraformGetDisabled: true,
		},
	}

	client := &getter.Client{
		// Src and Dst are intentioally left empty, they will be set before the call to client.Get()
		// because that's when these values are known.
		Src: "",
		Dst: "",

		Mode: getter.ClientModeAny,
		Pwd:  dir,
		Getters: map[string]getter.Getter{
			"git":   &getter.GitGetter{},
			"gcs":   &getter.GCSGetter{},
			"hg":    &getter.HgGetter{},
			"s3":    &getter.S3Getter{},
			"http":  httpGetter,
			"https": httpGetter,
		},
		Detectors: []getter.Detector{
			&getter.GitHubDetector{},
			&getter.GitLabDetector{},
			&getter.GitDetector{},
			&getter.BitBucketDetector{},
			&getter.S3Detector{},
			&getter.GCSDetector{},
			&getter.FileDetector{},
		},

		DisableSymlinks: true,
	}

	// Force client to have defaults configured
	client.Configure()
	return client
}

var getterURLRegexp = regexp.MustCompile(`^([A-Za-z0-9]+)::(.+)$`)

func extractProtocolFromURL(client *getter.Client, src string) (string, *url.URL, error) {
	if src == "" {
		// If empty we assume current directory and let NodeFile logic deal with finding and appropriate file
		u, err := url.Parse(".")
		return "file", u, err
	}

	parsed, err := getter.Detect(src, client.Pwd, client.Detectors)
	if err != nil {
		return "", nil, err
	}

	var proto string
	if ms := getterURLRegexp.FindStringSubmatch(parsed); ms != nil {
		proto = ms[1]
		parsed = ms[2]
	}

	u, err := url.Parse(parsed)
	if err != nil {
		return "", nil, err
	}

	if proto == "" {
		proto = u.Scheme
	}

	return proto, u, nil
}

func resolveHTTPEntrypoint(l *logger.Logger, insecure bool, timeout time.Duration, u *url.URL) (*url.URL, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return u, nil
	}

	if u.Scheme == "http" && !insecure {
		return nil, &errors.TaskfileNotSecureError{URI: u.String()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// HTTP and HTTPS entrypoint get filename auto-resolution before being handed off to go-getter
	u, err := RemoteExists(ctx, l, u)
	if err != nil {
		return nil, err
	} else if ctx.Err() != nil {
		return nil, &errors.TaskfileNetworkTimeoutError{URI: u.String(), Timeout: timeout}
	}

	return u, nil
}

func resolveTaskfileOverride(u *url.URL) (string, *url.URL) {
	q := u.Query()

	if f := q.Get("taskfile"); f != "" {
		// Delete magic parameter so we don't pass it onto further steps
		u2 := *u
		q.Del("taskfile")
		u2.RawQuery = q.Encode()

		return f, &u2
	}

	return "", u
}

// httpGetter wraps getter.HttpGetter to give us the ability
// to download single files into a directory, like other getters would
type httpGetter struct {
	*getter.HttpGetter
}

func (h httpGetter) Get(dst string, src *url.URL) error {
	// getter.HttpGetter does not support ClientModeDir for downloading directories (except when using the X-Terraform-Get feature, which we do not use).
	// By deferring to Get(), we allow this Getter to work in ClientModeDir while only downloading a single file.

	// In our case, dst is always a directory, so we append the filename to it.
	dst = filepath.Join(dst, filepath.Base(src.Path))

	return h.HttpGetter.GetFile(dst, src)
}

func (h httpGetter) ClientMode(*url.URL) (getter.ClientMode, error) {
	// We force ClientModeDir so we always get the same behaviour whether we're
	// requesting a file or a directory.  The override of Get() ensures this works
	// properly for both.
	return getter.ClientModeDir, nil
}
