package fsnotifyext

import (
	"math"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Deduper struct {
	w        *fsnotify.Watcher
	waitTime time.Duration
}

func NewDeduper(w *fsnotify.Watcher, waitTime time.Duration) *Deduper {
	return &Deduper{
		w:        w,
		waitTime: waitTime,
	}
}

// GetChan returns a chan of deduplicated [fsnotify.Event].
//
// [fsnotify.Chmod] operations will be skipped.
func (d *Deduper) GetChan() <-chan fsnotify.Event {
	channel := make(chan fsnotify.Event)

	go func() {
		timers := make(map[string]*time.Timer)
		for {
			event, ok := <-d.w.Events
			switch {
			case !ok:
				return
			case event.Has(fsnotify.Chmod):
				continue
			}

			timer, ok := timers[event.String()]
			if !ok {
				timer = time.AfterFunc(math.MaxInt64, func() { channel <- event })
				timer.Stop()
				timers[event.String()] = timer
			}

			timer.Reset(d.waitTime)
		}
	}()

	return channel
}
