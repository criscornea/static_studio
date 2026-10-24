package build

import (
	"sync"
)

// Event is one item in a build's output stream.
type Event struct {
	// Kind is "line" for output or "done" when the build has ended.
	Kind string `json:"kind"`
	Line string `json:"line,omitempty"`
	// OK and Error are set on "done" events.
	OK    bool   `json:"ok,omitempty"`
	Error string `json:"error,omitempty"`
	// Problems lists file errors parsed from the output. Set on "done" events.
	Problems []Problem `json:"problems,omitempty"`
}

const maxHistory = 10000

// Broadcaster fans events out to subscribers. It keeps the current build's
// events; each subscriber reads them through its own cursor, so a slow reader
// catches up instead of being dropped and never slows the build.
type Broadcaster struct {
	mu       sync.Mutex
	history  []Event
	finished bool
	epoch    int           // bumped by Reset; ends subscribers of older builds
	notify   chan struct{} // closed and replaced whenever history or epoch changes
}

// wake signals every waiting subscriber. The caller holds b.mu.
func (b *Broadcaster) wake() {
	if b.notify != nil {
		close(b.notify)
	}
	b.notify = make(chan struct{})
}

// Reset starts a new build: it drops the old history and ends any
// subscribers still attached to the previous one.
func (b *Broadcaster) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.history = nil
	b.finished = false
	b.epoch++
	b.wake()
}

// Publish records ev for every subscriber. A "done" event ends the stream:
// each subscriber channel is closed after delivering it.
func (b *Broadcaster) Publish(ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.history) < maxHistory || ev.Kind == "done" {
		b.history = append(b.history, ev)
	}
	if ev.Kind == "done" {
		b.finished = true
	}
	b.wake()
}

// Subscribe returns the events so far and a channel for later ones. The
// channel is closed when the build ends, is reset, or after cancel. Call
// cancel when done reading; it is safe to call more than once.
func (b *Broadcaster) Subscribe() (history []Event, ch <-chan Event, cancel func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	history = append([]Event(nil), b.history...)
	out := make(chan Event)
	if b.finished {
		close(out)
		return history, out, func() {}
	}
	if b.notify == nil {
		b.notify = make(chan struct{})
	}

	stop := make(chan struct{})
	var once sync.Once
	go b.pump(out, stop, len(b.history), b.epoch)

	return history, out, func() { once.Do(func() { close(stop) }) }
}

// pump delivers events from position pos onward until the build ends, is
// reset, or stop is closed.
func (b *Broadcaster) pump(out chan<- Event, stop <-chan struct{}, pos, epoch int) {
	defer close(out)

	for {
		b.mu.Lock()
		if b.epoch != epoch {
			b.mu.Unlock()
			return
		}
		if pos < len(b.history) {
			ev := b.history[pos]
			pos++
			b.mu.Unlock()

			select {
			case out <- ev:
			case <-stop:
				return
			}
			if ev.Kind == "done" {
				return
			}
			continue
		}
		wait := b.notify
		b.mu.Unlock()

		select {
		case <-wait:
		case <-stop:
			return
		}
	}
}
