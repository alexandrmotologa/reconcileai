package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEEvent represents a typed event dispatched to listening browser clients.
type SSEEvent struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// SSEHub manages concurrent subscriber channels for server-sent events.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]bool
}

// NewSSEHub initializes a new event broadcast hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan []byte]bool),
	}
}

// Subscribe adds a client channel to the active broadcast list.
func (h *SSEHub) Subscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[ch] = true
}

// Unsubscribe removes a client channel.
func (h *SSEHub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}

// Broadcast sends an event to all connected listeners.
func (h *SSEHub) Broadcast(eventType string, payload interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ev := SSEEvent{
		Type:      eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}

	data, err := json.Marshal(ev)
	if err != nil {
		return
	}

	msg := []byte(fmt.Sprintf("data: %s\n\n", data))
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
			// Non-blocking write to avoid lagging consumers
		}
	}
}

// HandleSSE handles incoming HTTP SSE subscription requests.
func (h *SSEHub) HandleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgChan := make(chan []byte, 16)
	h.Subscribe(msgChan)
	defer h.Unsubscribe(msgChan)

	// Send initial connection event
	initMsg := []byte(fmt.Sprintf("data: {\"type\":\"CONNECTED\",\"timestamp\":\"%s\"}\n\n", time.Now().UTC().Format(time.RFC3339)))
	w.Write(initMsg)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg := <-msgChan:
			w.Write(msg)
			flusher.Flush()
		}
	}
}
