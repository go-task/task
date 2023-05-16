package read

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

// An HTTPNode is a node that reads a Taskfile from a remote location via HTTP.
type HTTPNode struct {
	BaseNode
	Logger  *logger.Logger
	URL     *url.URL
	TempDir string
}

func NewHTTPNode(parent Node, urlString string, optional bool, tempDir string, l *logger.Logger) (*HTTPNode, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	return &HTTPNode{
		BaseNode: BaseNode{
			parent:   parent,
			optional: optional,
		},
		URL:     url,
		TempDir: tempDir,
		Logger:  l,
	}, nil
}

func (node *HTTPNode) Location() string {
	return node.URL.String()
}

func (node *HTTPNode) Read() (*taskfile.Taskfile, error) {
	resp, err := http.Get(node.URL.String())
	if err != nil {
		return nil, errors.TaskfileNotFoundError{URI: node.URL.String()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.TaskfileNotFoundError{URI: node.URL.String()}
	}

	// Read the entire response body
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Create a hash of the response body
	h := sha256.New()
	h.Write(b)
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Get the cached hash
	cachedHash, err := node.getHashFromCache()
	if errors.Is(err, os.ErrNotExist) {
		// If the hash doesn't exist in the cache, prompt the user to continue
		if cont, err := node.Logger.Prompt(logger.Yellow, fmt.Sprintf("The task you are attempting to run depends on the remote Taskfile at %q.\n--- Make sure you trust the source of this Taskfile before continuing ---\nContinue?", node.URL.String()), "n", "y", "yes"); err != nil {
			return nil, err
		} else if !cont {
			return nil, &errors.TaskfileNotTrustedError{URI: node.URL.String()}
		}
	} else if err != nil {
		return nil, err
	} else if hash != cachedHash {
		// If there is a cached hash, but it doesn't match the expected hash, prompt the user to continue
		if cont, err := node.Logger.Prompt(logger.Yellow, fmt.Sprintf("The Taskfile at %q has changed since you last used it!\n--- Make sure you trust the source of this Taskfile before continuing ---\nContinue?", node.URL.String()), "n", "y", "yes"); err != nil {
			return nil, err
		} else if !cont {
			return nil, &errors.TaskfileNotTrustedError{URI: node.URL.String()}
		}
	}

	// If the hash has changed (or is new), store it in the cache
	if hash != cachedHash {
		if err := node.toCache([]byte(hash)); err != nil {
			return nil, err
		}
	}

	// Unmarshal the taskfile
	var t *taskfile.Taskfile
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, &errors.TaskfileInvalidError{URI: node.URL.String(), Err: err}
	}
	t.Location = node.URL.String()
	return t, nil
}

func (node *HTTPNode) getCachePath() string {
	h := sha256.New()
	h.Write([]byte(node.URL.String()))
	return filepath.Join(node.TempDir, base64.URLEncoding.EncodeToString(h.Sum(nil)))
}

func (node *HTTPNode) getHashFromCache() (string, error) {
	path := node.getCachePath()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (node *HTTPNode) toCache(hash []byte) error {
	path := node.getCachePath()
	return os.WriteFile(path, hash, 0o644)
}
