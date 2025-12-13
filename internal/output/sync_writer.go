package output

import (
	"io"
	"sync"
)

// SyncWriter wraps an io.Writer with a mutex to synchronize writes.
// This is used to prevent output from interleaving with interactive prompts.
type SyncWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

// NewSyncWriter creates a new SyncWriter that uses the provided mutex.
func NewSyncWriter(w io.Writer, mu *sync.Mutex) *SyncWriter {
	return &SyncWriter{
		w:  w,
		mu: mu,
	}
}

// Write implements io.Writer with synchronized access.
func (sw *SyncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}
