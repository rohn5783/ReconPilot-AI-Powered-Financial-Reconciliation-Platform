package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"reconpilot/internal/repository"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Configurable per CORS in router
	},
}

// StreamHandler manages real-time WebSocket and SSE connections for scan telemetry.
type StreamHandler struct {
	hub *repository.RedisTelemetryHub
}

// NewStreamHandler creates a new StreamHandler.
func NewStreamHandler(hub *repository.RedisTelemetryHub) *StreamHandler {
	return &StreamHandler{hub: hub}
}

// HandleWebSocket upgrades HTTP to WebSocket and streams real-time ANSI terminal logs & events.
func (h *StreamHandler) HandleWebSocket(c *gin.Context) {
	scanIDStr := c.Param("id")
	scanJobID, err := uuid.Parse(scanIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan job UUID"})
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	eventChan, unsubscribe := h.hub.Subscribe(c.Request.Context(), scanJobID)
	defer unsubscribe()

	// Initial connection handshake frame
	initMsg := map[string]interface{}{
		"type":        "connected",
		"scan_job_id": scanJobID.String(),
		"timestamp":   time.Now().Format(time.RFC3339),
	}
	initJSON, _ := json.Marshal(initMsg)
	_ = ws.WriteMessage(websocket.TextMessage, initJSON)

	// Keepalive ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Reader pump to detect client close
	go func() {
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}()

	for {
		select {
		case ev, ok := <-eventChan:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}

		case <-c.Request.Context().Done():
			return
		}
	}
}

// HandleSSE streams telemetry via Server-Sent Events (SSE) for HTTP clients.
func (h *StreamHandler) HandleSSE(c *gin.Context) {
	scanIDStr := c.Param("id")
	scanJobID, err := uuid.Parse(scanIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan job UUID"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	eventChan, unsubscribe := h.hub.Subscribe(c.Request.Context(), scanJobID)
	defer unsubscribe()

	c.Stream(func(w io.Writer) bool {
		select {
		case ev, ok := <-eventChan:
			if !ok {
				return false
			}
			data, err := json.Marshal(ev)
			if err != nil {
				return true
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, string(data))
			return true

		case <-c.Request.Context().Done():
			return false
		}
	})
}

