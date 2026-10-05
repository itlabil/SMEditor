package job

import (
	"sync"
	"time"
)

// Hub fans job Events out to every SSE client subscribed to a project, per
// .agents/skills/sm-job-worker. It never blocks the worker: progress
// events are dropped if a subscriber's buffer is full, but terminal events
// (done, failed, canceled) are always eventually delivered.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan Event]struct{}{}}
}

// Subscribe returns a channel of events for projectID. Call unsubscribe
// when the client disconnects. The channel is never closed (only
// dereferenced), so a Publish racing with unsubscribe can never send on a
// closed channel.
func (h *Hub) Subscribe(projectID string) (ch chan Event, unsubscribe func()) {
	ch = make(chan Event, 16)
	h.mu.Lock()
	if h.subs[projectID] == nil {
		h.subs[projectID] = map[chan Event]struct{}{}
	}
	h.subs[projectID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs[projectID], ch)
		if len(h.subs[projectID]) == 0 {
			delete(h.subs, projectID)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) Publish(projectID string, ev Event) {
	h.mu.Lock()
	chans := make([]chan Event, 0, len(h.subs[projectID]))
	for ch := range h.subs[projectID] {
		chans = append(chans, ch)
	}
	h.mu.Unlock()

	terminal := ev.Type != "progress"
	for _, ch := range chans {
		select {
		case ch <- ev:
		default:
			if terminal {
				go deliverWithTimeout(ch, ev)
			}
		}
	}
}

func deliverWithTimeout(ch chan Event, ev Event) {
	select {
	case ch <- ev:
	case <-time.After(5 * time.Second):
	}
}
