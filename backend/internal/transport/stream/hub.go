package stream

import (
	"sync"
	"time"

	"reconpilot/internal/domain"
)

// Hub manages real-time broadcast of pipeline telemetry events to SSE and WebSocket clients.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan domain.PipelineTelemetryEvent]bool
	history     []domain.PipelineTelemetryEvent
	maxHistory  int
}

// NewHub creates an event broadcaster with bounded history.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[chan domain.PipelineTelemetryEvent]bool),
		history:     make([]domain.PipelineTelemetryEvent, 0, 100),
		maxHistory:  100,
	}
}

// Subscribe registers a new telemetry subscriber channel and returns an unsubscribe function.
func (h *Hub) Subscribe() (<-chan domain.PipelineTelemetryEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan domain.PipelineTelemetryEvent, 64)
	h.subscribers[ch] = true

	// Send recent history to new subscriber
	for _, evt := range h.history {
		select {
		case ch <- evt:
		default:
		}
	}

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}
	}

	return ch, unsubscribe
}

// Broadcast distributes a telemetry event to all active subscribers.
func (h *Hub) Broadcast(evt domain.PipelineTelemetryEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	if len(h.history) >= h.maxHistory {
		h.history = h.history[1:]
	}
	h.history = append(h.history, evt)

	for ch := range h.subscribers {
		select {
		case ch <- evt:
		default:
			// Non-blocking drop if consumer buffer is full
		}
	}
}

// History returns a copy of recent events.
func (h *Hub) History() []domain.PipelineTelemetryEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	res := make([]domain.PipelineTelemetryEvent, len(h.history))
	copy(res, h.history)
	return res
}

