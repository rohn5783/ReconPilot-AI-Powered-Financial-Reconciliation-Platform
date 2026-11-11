package stream

import (
	"testing"
	"time"

	"reconpilot/internal/domain"
)

func TestHubBroadcastAndSubscribe(t *testing.T) {
	hub := NewHub()

	ch, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	event := domain.PipelineTelemetryEvent{
		Type:               "progress",
		Stage:              "LEVEL_1_MATCH",
		ProgressPercentage: 50.0,
		Message:            "Test broadcast message",
		Timestamp:          time.Now().UTC(),
	}

	hub.Broadcast(event)

	select {
	case received := <-ch:
		if received.Message != event.Message {
			t.Errorf("Expected message %q, got %q", event.Message, received.Message)
		}
		if received.Stage != "LEVEL_1_MATCH" {
			t.Errorf("Expected stage LEVEL_1_MATCH, got %q", received.Stage)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for telemetry broadcast event")
	}

	history := hub.History()
	if len(history) != 1 {
		t.Errorf("Expected 1 history event, got %d", len(history))
	}
}
