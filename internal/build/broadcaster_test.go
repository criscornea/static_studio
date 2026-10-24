package build

import (
	"strconv"
	"testing"
)

func line(s string) Event { return Event{Kind: "line", Line: s} }

func drain(ch <-chan Event) []Event {
	var out []Event
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

func TestBroadcastFanOut(t *testing.T) {
	var b Broadcaster
	_, c1, _ := b.Subscribe()
	_, c2, _ := b.Subscribe()

	b.Publish(line("a"))
	b.Publish(Event{Kind: "done", OK: true})

	for _, c := range []<-chan Event{c1, c2} {
		got := drain(c)
		if len(got) != 2 || got[0].Line != "a" || got[1].Kind != "done" {
			t.Errorf("got %+v", got)
		}
	}
}

func TestBroadcastLateSubscriberReplay(t *testing.T) {
	var b Broadcaster
	b.Publish(line("a"))

	hist, ch, cancel := b.Subscribe()
	defer cancel()
	if len(hist) != 1 || hist[0].Line != "a" {
		t.Fatalf("hist = %+v", hist)
	}

	b.Publish(line("b"))
	if ev := <-ch; ev.Line != "b" {
		t.Errorf("ev = %+v", ev)
	}

	b.Publish(Event{Kind: "done"})
	hist, ch, _ = b.Subscribe()
	if len(hist) != 3 {
		t.Errorf("hist after done = %+v", hist)
	}
	if _, open := <-ch; open {
		t.Error("channel should be closed after done")
	}
}

func TestBroadcastCancelAndReset(t *testing.T) {
	var b Broadcaster
	_, ch, cancel := b.Subscribe()
	cancel()
	cancel()
	if _, open := <-ch; open {
		t.Error("cancel should close channel")
	}

	b.Publish(line("a"))
	_, ch2, _ := b.Subscribe()
	b.Reset()
	if _, open := <-ch2; open {
		t.Error("reset should close subscribers")
	}
	if hist, _, _ := b.Subscribe(); len(hist) != 0 {
		t.Errorf("hist = %+v", hist)
	}
}

func TestBroadcastSlowSubscriberCatchesUp(t *testing.T) {
	var b Broadcaster
	_, ch, cancel := b.Subscribe()
	defer cancel()

	const n = 1000
	for i := range n {
		b.Publish(line(strconv.Itoa(i)))
	}
	b.Publish(Event{Kind: "done", OK: true})

	got := drain(ch)
	if len(got) != n+1 {
		t.Fatalf("received %d events, want %d", len(got), n+1)
	}
	if got[n].Kind != "done" || got[n-1].Line != strconv.Itoa(n-1) {
		t.Errorf("tail = %+v %+v", got[n-1], got[n])
	}
}
