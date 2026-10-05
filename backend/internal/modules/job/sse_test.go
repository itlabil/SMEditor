package job

import (
	"testing"
	"time"
)

func TestHubDeliversEventToSubscriber(t *testing.T) {
	hub := NewHub()
	ch, unsubscribe := hub.Subscribe("proj-x")
	defer unsubscribe()

	hub.Publish("proj-x", Event{Type: "progress", JobID: "j1", Progress: 10})

	select {
	case ev := <-ch:
		if ev.JobID != "j1" {
			t.Errorf("JobID = %q, want j1", ev.JobID)
		}
	case <-time.After(time.Second):
		t.Fatal("event never delivered")
	}
}

func TestHubDropsProgressWhenBufferFull(t *testing.T) {
	hub := NewHub()
	ch, unsubscribe := hub.Subscribe("proj-y")
	defer unsubscribe()

	// Fill the buffer (16) without draining, then publish one more: it
	// must be dropped silently, not block.
	for i := 0; i < 20; i++ {
		hub.Publish("proj-y", Event{Type: "progress", Progress: float64(i)})
	}

	if len(ch) == 0 {
		t.Fatal("expected buffered events, got none")
	}
}

func TestHubDeliversTerminalEventEvenWhenBufferFull(t *testing.T) {
	hub := NewHub()
	ch, unsubscribe := hub.Subscribe("proj-z")
	defer unsubscribe()

	for i := 0; i < 20; i++ {
		hub.Publish("proj-z", Event{Type: "progress", Progress: float64(i)})
	}
	hub.Publish("proj-z", Event{Type: "done", JobID: "final"})

	deadline := time.After(6 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type == "done" && ev.JobID == "final" {
				return
			}
		case <-deadline:
			t.Fatal("terminal event was never delivered")
		}
	}
}

func TestHubUnsubscribeThenPublishDoesNotPanic(t *testing.T) {
	hub := NewHub()
	_, unsubscribe := hub.Subscribe("proj-w")
	unsubscribe()

	hub.Publish("proj-w", Event{Type: "done"})
}
