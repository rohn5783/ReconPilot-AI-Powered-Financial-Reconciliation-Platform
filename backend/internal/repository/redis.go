package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"reconpilot/internal/config"
	"reconpilot/internal/domain"
)

// RedisTelemetryHub handles real-time telemetry distribution via Redis Pub/Sub,
// with automatic fallback to high-performance local memory channels if Redis is unavailable.
type RedisTelemetryHub struct {
	client    *redis.Client
	isLive    bool
	mu        sync.RWMutex
	localSubs map[uuid.UUID][]chan domain.TelemetryEvent
}

// NewRedisTelemetryHub initializes the Redis client or local fallback.
func NewRedisTelemetryHub(ctx context.Context, cfg *config.RedisConfig) *RedisTelemetryHub {
	if ctx == nil {
		ctx = context.Background()
	}
	hub := &RedisTelemetryHub{
		localSubs: make(map[uuid.UUID][]chan domain.TelemetryEvent),
	}

	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		opts = &redis.Options{
			Addr: "localhost:6379",
		}
	}
	if cfg.Password != "" {
		opts.Password = cfg.Password
	}
	if cfg.DB != 0 {
		opts.DB = cfg.DB
	}

	rdb := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err == nil {
		hub.client = rdb
		hub.isLive = true
	}

	return hub
}

// Broadcast distributes a telemetry event to all active Redis and in-memory subscribers.
func (h *RedisTelemetryHub) Broadcast(ctx context.Context, event domain.TelemetryEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal telemetry event: %w", err)
	}

	// 1. Publish to Redis if connected
	if h.isLive && h.client != nil {
		channel := fmt.Sprintf("reconpilot:telemetry:%s", event.ScanJobID.String())
		_ = h.client.Publish(ctx, channel, eventJSON).Err()
	}

	// 2. Dispatch to local subscribers
	h.mu.RLock()
	subs, exists := h.localSubs[event.ScanJobID]
	h.mu.RUnlock()

	if exists {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
				// Non-blocking drop if channel buffer full
			}
		}
	}

	return nil
}

// Subscribe registers a listener for a specific scan job's real-time events.
func (h *RedisTelemetryHub) Subscribe(ctx context.Context, scanJobID uuid.UUID) (<-chan domain.TelemetryEvent, func()) {
	out := make(chan domain.TelemetryEvent, 256)

	// Register in local registry
	h.mu.Lock()
	h.localSubs[scanJobID] = append(h.localSubs[scanJobID], out)
	h.mu.Unlock()

	// If Redis is active, start a background goroutine relaying Redis pub/sub to out channel
	var pubsub *redis.PubSub
	if h.isLive && h.client != nil {
		channel := fmt.Sprintf("reconpilot:telemetry:%s", scanJobID.String())
		pubsub = h.client.Subscribe(ctx, channel)

		go func() {
			ch := pubsub.Channel()
			for msg := range ch {
				var ev domain.TelemetryEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err == nil {
					select {
					case out <- ev:
					default:
					}
				}
			}
		}()
	}

	cleanup := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if pubsub != nil {
			_ = pubsub.Close()
		}

		subs := h.localSubs[scanJobID]
		for i, ch := range subs {
			if ch == out {
				h.localSubs[scanJobID] = append(subs[:i], subs[i+1:]...)
				close(out)
				break
			}
		}
	}

	return out, cleanup
}

