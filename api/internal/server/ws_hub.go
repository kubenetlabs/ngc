package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage is a message sent over WebSocket to clients.
type WSMessage struct {
	Topic     string          `json:"topic"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// wsClient represents a single WebSocket connection.
type wsClient struct {
	conn   *websocket.Conn
	topics map[string]bool
	send   chan []byte
}

// Hub manages WebSocket clients and topic-based broadcasting.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*wsClient]bool
	register   chan *wsClient
	unregister chan *wsClient
	generators []topicGenerator
	stopCh     chan struct{}
}

type topicGenerator struct {
	topic    string
	interval time.Duration
	generate func() (json.RawMessage, error)
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*wsClient]bool),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		stopCh:     make(chan struct{}),
	}
}

// AddGenerator registers a topic generator that periodically produces data.
func (h *Hub) AddGenerator(topic string, interval time.Duration, fn func() (json.RawMessage, error)) {
	h.generators = append(h.generators, topicGenerator{
		topic:    topic,
		interval: interval,
		generate: fn,
	})
}

// Start begins the hub's run loop and all registered generators.
func (h *Hub) Start() {
	go h.run()
	for _, gen := range h.generators {
		go h.runGenerator(gen)
	}
}

// Stop shuts down the hub and disconnects all clients.
func (h *Hub) Stop() {
	close(h.stopCh)
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Info("ws client registered", "topics", client.topics)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case <-h.stopCh:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

func (h *Hub) broadcast(topic string, data json.RawMessage) {
	msg := WSMessage{
		Topic:     topic,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		slog.Error("ws marshal error", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.topics[topic] || client.topics["*"] {
			select {
			case client.send <- payload:
			default:
				// Client too slow, drop message
			}
		}
	}
}

func (h *Hub) runGenerator(gen topicGenerator) {
	ticker := time.NewTicker(gen.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			data, err := gen.generate()
			if err != nil {
				slog.Error("ws generator error", "topic", gen.topic, "error", err)
				continue
			}
			h.broadcast(gen.topic, data)
		case <-h.stopCh:
			return
		}
	}
}

// ServeWS returns an HTTP handler that upgrades connections to WebSocket
// and subscribes them to the given topic.
func (h *Hub) ServeWS(topic string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			return
		}

		client := &wsClient{
			conn:   conn,
			topics: map[string]bool{topic: true},
			send:   make(chan []byte, 64),
		}
		h.register <- client

		go h.writePump(client)
		go h.readPump(client)
	}
}

func (h *Hub) writePump(client *wsClient) {
	defer client.conn.Close()
	for msg := range client.send {
		if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (h *Hub) readPump(client *wsClient) {
	defer func() {
		h.unregister <- client
		client.conn.Close()
	}()
	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
