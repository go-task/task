package fsnotifyext

import (
	"math"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Deduper struct {
	w        *fsnotify.Watcher
	waitTime time.Duration
	mutex    sync.Mutex
}

func NewDeduper(w *fsnotify.Watcher, waitTime time.Duration) *Deduper {
	return &Deduper{
		w:        w,
		waitTime: waitTime,
	}
}

func (d *Deduper) GetChan() chan fsnotify.Event {
	channel := make(chan fsnotify.Event)
	timers := make(map[string]*time.Timer)

	go func() {
		for {
			event, ok := <-d.w.Events
			switch {
			case !ok:
				return
			case event.Op == fsnotify.Chmod:
				continue
			}

			d.mutex.Lock()
			timer, ok := timers[event.String()]
			d.mutex.Unlock()

			if !ok {
				timer = time.AfterFunc(math.MaxInt64, func() { channel <- event })
				timer.Stop()

				d.mutex.Lock()
				timers[event.String()] = timer
				d.mutex.Unlock()
			}

			timer.Reset(d.waitTime)
		}
	}()

	return channel
}
