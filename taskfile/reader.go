package taskfile

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/dominikbraun/graph"
	"go.yaml.in/yaml/v4"
	"golang.org/x/sync/errgroup"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

const (
	taskfileUntrustedPrompt = `The task you are attempting to run depends on the remote Taskfile at %q.
--- Make sure you trust the source of this Taskfile before continuing ---
Continue?`
	taskfileChangedPrompt = `The Taskfile at %q has changed since you last used it!
--- Make sure you trust the source of this Taskfile before continuing ---
Continue?`
)

type (
	// DebugFunc is a function that can be called to log debug messages.
	DebugFunc func(string)
	// PromptFunc is a function that can be called to prompt the user for input.
	PromptFunc func(string) error
	// A ReaderOption is any type that can apply a configuration to a [Reader].
	ReaderOption interface {
		ApplyToReader(*Reader)
	}
	// A Reader will recursively read Taskfiles from a given [Node] and build a
	// [ast.TaskfileGraph] from them.
	Reader struct {
		graph               *ast.TaskfileGraph
		insecure            bool
		download            bool
		offline             bool
		trustedHosts        []string
		tempDir             string
		cacheExpiryDuration time.Duration
		debugFunc           DebugFunc
		promptFunc          PromptFunc
		promptMutex         sync.Mutex
	}
)

// NewReader constructs a new Taskfile [Reader] using the given Node and
// options.
func NewReader(opts ...ReaderOption) *Reader {
	r := &Reader{
		graph:               ast.NewTaskfileGraph(),
		insecure:            false,
		download:            false,
		offline:             false,
		trustedHosts:        nil,
		tempDir:             os.TempDir(),
		cacheExpiryDuration: 0,
		debugFunc:           nil,
		promptFunc:          nil,
		promptMutex:         sync.Mutex{},
	}
	r.Options(opts...)
	return r
}

// Options loops through the given [ReaderOption] functions and applies them to
// the [Reader].
func (r *Reader) Options(opts ...ReaderOption) {
	for _, opt := range opts {
		opt.ApplyToReader(r)
	}
}

// WithInsecure allows the [Reader] to make insecure connections when reading
// remote taskfiles. By default, insecure connections are rejected.
func WithInsecure(insecure bool) ReaderOption {
	return &insecureOption{insecure: insecure}
}

type insecureOption struct {
	insecure bool
}

func (o *insecureOption) ApplyToReader(r *Reader) {
	r.insecure = o.insecure
}

// WithDownload forces the [Reader] to download a fresh copy of the taskfile
// from the remote source.
func WithDownload(download bool) ReaderOption {
	return &downloadOption{download: download}
}

type downloadOption struct {
	download bool
}

func (o *downloadOption) ApplyToReader(r *Reader) {
	r.download = o.download
}

// WithOffline stops the [Reader] from being able to make network connections.
// It will still be able to read local files and cached copies of remote files.
func WithOffline(offline bool) ReaderOption {
	return &offlineOption{offline: offline}
}

type offlineOption struct {
	offline bool
}

func (o *offlineOption) ApplyToReader(r *Reader) {
	r.offline = o.offline
}

// WithTrustedHosts configures the [Reader] with a list of trusted hosts for remote
// Taskfiles. Hosts in this list will not prompt for user confirmation.
func WithTrustedHosts(trustedHosts []string) ReaderOption {
	return &trustedHostsOption{trustedHosts: trustedHosts}
}

type trustedHostsOption struct {
	trustedHosts []string
}

func (o *trustedHostsOption) ApplyToReader(r *Reader) {
	r.trustedHosts = o.trustedHosts
}

// WithTempDir sets the temporary directory that will be used by the [Reader].
// By default, the reader uses [os.TempDir].
func WithTempDir(tempDir string) ReaderOption {
	return &tempDirOption{tempDir: tempDir}
}

type tempDirOption struct {
	tempDir string
}

func (o *tempDirOption) ApplyToReader(r *Reader) {
	r.tempDir = o.tempDir
}

// WithCacheExpiryDuration sets the duration after which the cache is considered
// expired. By default, the cache is considered expired after 24 hours.
func WithCacheExpiryDuration(duration time.Duration) ReaderOption {
	return &cacheExpiryDurationOption{duration: duration}
}

type cacheExpiryDurationOption struct {
	duration time.Duration
}

func (o *cacheExpiryDurationOption) ApplyToReader(r *Reader) {
	r.cacheExpiryDuration = o.duration
}

// WithDebugFunc sets the debug function to be used by the [Reader]. If set,
// this function will be called with debug messages. This can be useful if the
// caller wants to log debug messages from the [Reader]. By default, no debug
// function is set and the logs are not written.
func WithDebugFunc(debugFunc DebugFunc) ReaderOption {
	return &debugFuncOption{debugFunc: debugFunc}
}

type debugFuncOption struct {
	debugFunc DebugFunc
}

func (o *debugFuncOption) ApplyToReader(r *Reader) {
	r.debugFunc = o.debugFunc
}

// WithPromptFunc sets the prompt function to be used by the [Reader]. If set,
// this function will be called with prompt messages. The function should
// optionally log the message to the user and return nil if the prompt is
// accepted and the execution should continue. Otherwise, it should return an
// error which describes why the prompt was rejected. This can then be caught
// and used later when calling the [Reader.Read] method. By default, no prompt
// function is set and all prompts are automatically accepted.
func WithPromptFunc(promptFunc PromptFunc) ReaderOption {
	return &promptFuncOption{promptFunc: promptFunc}
}

type promptFuncOption struct {
	promptFunc PromptFunc
}

func (o *promptFuncOption) ApplyToReader(r *Reader) {
	r.promptFunc = o.promptFunc
}

// Read will read the Taskfile defined by the [Reader]'s [Node] and recurse
// through any [ast.Includes] it finds, reading each included Taskfile and
// building an [ast.TaskfileGraph] as it goes. If any errors occur, they will be
// returned immediately.
func (r *Reader) Read(ctx context.Context, node Node) (*ast.TaskfileGraph, error) {
	// Clean up git cache after reading all taskfiles
	defer func() {
		_ = CleanGitCache()
	}()

	if err := r.include(ctx, node); err != nil {
		return nil, err
	}

	return r.graph, nil
}

func (r *Reader) debugf(format string, a ...any) {
	if r.debugFunc != nil {
		r.debugFunc(fmt.Sprintf(format, a...))
	}
}

func (r *Reader) promptf(format string, a ...any) error {
	if r.promptFunc != nil {
		return r.promptFunc(fmt.Sprintf(format, a...))
	}
	return nil
}

// isTrusted checks if a URI's host matches any of the trusted hosts patterns.
func (r *Reader) isTrusted(uri string) bool {
	if len(r.trustedHosts) == 0 {
		return false
	}

	// Parse the URI to extract the host
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return false
	}
	host := parsedURL.Host

	// Check against each trusted pattern (exact match including port if provided)
	for _, pattern := range r.trustedHosts {
		if host == pattern {
			return true
		}
	}
	return false
}

func (r *Reader) include(ctx context.Context, node Node) error {
	// Create a new vertex for the Taskfile
	vertex := &ast.TaskfileVertex{
		URI:      node.Location(),
		Taskfile: nil,
	}

	// Add the included Taskfile to the DAG
	// If the vertex already exists, we return early since its Taskfile has
	// already been read and its children explored
	if err := r.graph.AddVertex(vertex); err == graph.ErrVertexAlreadyExists {
		return nil
	} else if err != nil {
		return err
	}

	// Read and parse the Taskfile from the file and add it to the vertex
	var err error
	vertex.Taskfile, err = r.readNode(ctx, node)
	if err != nil {
		return err
	}

	// Create an error group to wait for all included Taskfiles to be read
	var g errgroup.Group

	// Loop over each included taskfile
	for _, include := range vertex.Taskfile.Includes.All() {
		vars := env.GetEnviron()
		vars.Merge(vertex.Taskfile.Vars, nil)
		// Start a goroutine to process each included Taskfile
		g.Go(func() error {
			cache := &templater.Cache{Vars: vars}
			include = &ast.Include{
				Namespace:      include.Namespace,
				Taskfile:       templater.Replace(include.Taskfile, cache),
				Dir:            templater.Replace(include.Dir, cache),
				Optional:       include.Optional,
				Internal:       include.Internal,
				Flatten:        include.Flatten,
				Aliases:        include.Aliases,
				AdvancedImport: include.AdvancedImport,
				Excludes:       include.Excludes,
				Vars:           include.Vars,
				Checksum:       include.Checksum,
			}
			if err := cache.Err(); err != nil {
				return err
			}

			entrypoint, err := node.ResolveEntrypoint(include.Taskfile)
			if err != nil {
				return err
			}

			include.Dir, err = node.ResolveDir(include.Dir)
			if err != nil {
				return err
			}

			includeNode, err := NewNode(entrypoint, include.Dir, r.insecure,
				WithParent(node),
				WithChecksum(include.Checksum),
			)
			if err != nil {
				if include.Optional {
					return nil
				}
				return err
			}

			// Recurse into the included Taskfile
			if err := r.include(ctx, includeNode); err != nil {
				return err
			}

			// Create an edge between the Taskfiles
			r.graph.Lock()
			defer r.graph.Unlock()
			edge, err := r.graph.Edge(node.Location(), includeNode.Location())
			if err == graph.ErrEdgeNotFound {
				// If the edge doesn't exist, create it
				err = r.graph.AddEdge(
					node.Location(),
					includeNode.Location(),
					graph.EdgeData([]*ast.Include{include}),
					graph.EdgeWeight(1),
				)
			} else {
				// If the edge already exists
				edgeData := append(edge.Properties.Data.([]*ast.Include), include)
				err = r.graph.UpdateEdge(
					node.Location(),
					includeNode.Location(),
					graph.EdgeData(edgeData),
					graph.EdgeWeight(len(edgeData)),
				)
			}
			if errors.Is(err, graph.ErrEdgeCreatesCycle) {
				return errors.TaskfileCycleError{
					Source:      node.Location(),
					Destination: includeNode.Location(),
				}
			}
			return err
		})
	}

	// Wait for all the go routines to finish
	return g.Wait()
}

func (r *Reader) readNode(ctx context.Context, node Node) (*ast.Taskfile, error) {
	b, err := r.readNodeContent(ctx, node)
	if err != nil {
		return nil, err
	}

	var tf ast.Taskfile
	if err := yaml.Unmarshal(b, &tf); err != nil {
		// Decode the taskfile and add the file info the any errors
		taskfileDecodeErr := &errors.TaskfileDecodeError{}
		if errors.As(err, &taskfileDecodeErr) {
			snippet := NewSnippet(b,
				WithLine(taskfileDecodeErr.Line),
				WithColumn(taskfileDecodeErr.Column),
				WithPadding(2),
			)
			return nil, taskfileDecodeErr.WithFileInfo(node.Location(), snippet.String())
		}
		return nil, &errors.TaskfileInvalidError{URI: filepathext.TryAbsToRel(node.Location()), Err: err}
	}

	// Check that the Taskfile is set and has a schema version
	if tf.Version == nil {
		return nil, &errors.TaskfileVersionCheckError{URI: node.Location()}
	}

	// Set the taskfile/task's locations
	tf.Location = node.Location()
	for task := range tf.Tasks.Values(nil) {
		// If the task is not defined, create a new one
		if task == nil {
			task = &ast.Task{}
		}
		// Set the location of the taskfile for each task
		if task.Location.Taskfile == "" {
			task.Location.Taskfile = tf.Location
		}
	}

	return &tf, nil
}

func (r *Reader) readNodeContent(ctx context.Context, node Node) ([]byte, error) {
	if node, isRemote := node.(RemoteNode); isRemote {
		return r.readRemoteNodeContent(ctx, node)
	}

	// Read the Taskfile
	b, err := node.Read()
	if err != nil {
		return nil, err
	}

	// If the given checksum doesn't match the sum pinned in the Taskfile
	checksum := checksum(b)
	if !node.Verify(checksum) {
		return nil, &errors.TaskfileDoesNotMatchChecksum{
			URI:              node.Location(),
			ExpectedChecksum: node.Checksum(),
			ActualChecksum:   checksum,
		}
	}

	return b, nil
}

func (r *Reader) readRemoteNodeContent(ctx context.Context, node RemoteNode) ([]byte, error) {
	cache := NewCacheNode(node, r.tempDir)
	now := time.Now().UTC()
	timestamp := cache.ReadTimestamp()
	expiry := timestamp.Add(r.cacheExpiryDuration)
	cacheValid := now.Before(expiry)
	var cacheFound bool

	r.debugf("checking cache for %q in %q\n", node.Location(), cache.Location())
	cachedBytes, err := cache.Read()
	switch {
	// If the cache doesn't exist, we need to download the file
	case errors.Is(err, os.ErrNotExist):
		r.debugf("no cache found\n")
		// If we couldn't find a cached copy, and we are offline, we can't do anything
		if r.offline {
			return nil, &errors.TaskfileCacheNotFoundError{
				URI: node.Location(),
			}
		}

	// If the cache is expired
	case !cacheValid:
		r.debugf("cache expired at %s\n", expiry.Format(time.RFC3339))
		cacheFound = true
		// If we can't fetch a fresh copy, we should use the cache anyway
		if r.offline {
			r.debugf("in offline mode, using expired cache\n")
			return cachedBytes, nil
		}

	// Some other error
	case err != nil:
		return nil, err

	// Found valid cache
	default:
		r.debugf("cache found\n")
		// Not being forced to redownload, return cache
		if !r.download {
			return cachedBytes, nil
		}
		cacheFound = true
	}

	// Try to read the remote file
	r.debugf("downloading remote file: %s\n", node.Location())
	downloadedBytes, err := node.ReadContext(ctx)
	if err != nil {
		// If the context timed out or was cancelled, but we found a cached version, use that
		if ctx.Err() != nil && cacheFound {
			if cacheValid {
				r.debugf("failed to fetch remote file: %s: using cache\n", ctx.Err().Error())
			} else {
				r.debugf("failed to fetch remote file: %s: using expired cache\n", ctx.Err().Error())
			}
			return cachedBytes, nil
		}
		return nil, err
	}

	r.debugf("found remote file at %q\n", node.Location())

	// If the given checksum doesn't match the sum pinned in the Taskfile
	checksum := checksum(downloadedBytes)
	if !node.Verify(checksum) {
		return nil, &errors.TaskfileDoesNotMatchChecksum{
			URI:              node.Location(),
			ExpectedChecksum: node.Checksum(),
			ActualChecksum:   checksum,
		}
	}

	// If there is no manual checksum pin, run the automatic checks
	if node.Checksum() == "" {
		// Prompt the user if required (unless host is trusted)
		prompt := cache.ChecksumPrompt(checksum)
		if prompt != "" && !r.isTrusted(node.Location()) {
			if err := func() error {
				r.promptMutex.Lock()
				defer r.promptMutex.Unlock()
				return r.promptf(prompt, node.Location())
			}(); err != nil {
				return nil, &errors.TaskfileNotTrustedError{URI: node.Location()}
			}
		}
	}

	// Store the checksum
	if err := cache.WriteChecksum(checksum); err != nil {
		return nil, err
	}

	// Store the timestamp
	if err := cache.WriteTimestamp(now); err != nil {
		return nil, err
	}

	// Cache the file
	r.debugf("caching %q to %q\n", node.Location(), cache.Location())
	if err = cache.Write(downloadedBytes); err != nil {
		return nil, err
	}

	return downloadedBytes, nil
}
