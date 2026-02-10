package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.clients == nil {
		t.Error("expected initialized clients map")
	}
	if hub.register == nil {
		t.Error("expected initialized register channel")
	}
	if hub.unregister == nil {
		t.Error("expected initialized unregister channel")
	}
}

func TestHub_AddGenerator(t *testing.T) {
	hub := NewHub()
	hub.AddGenerator("test-topic", time.Second, func() (json.RawMessage, error) {
		return json.Marshal(map[string]string{"hello": "world"})
	})
	if len(hub.generators) != 1 {
		t.Fatalf("expected 1 generator, got %d", len(hub.generators))
	}
	if hub.generators[0].topic != "test-topic" {
		t.Errorf("expected topic test-topic, got %s", hub.generators[0].topic)
	}
}

func TestHub_BroadcastToSubscribedClient(t *testing.T) {
	hub := NewHub()
	hub.Start()
	defer hub.Stop()

	// Set up a test WebSocket server
	s := httptest.NewServer(http.HandlerFunc(hub.ServeWS("test-topic")))
	defer s.Close()

	// Connect a WebSocket client
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Give the hub time to register the client
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message
	data, _ := json.Marshal(map[string]string{"key": "value"})
	hub.broadcast("test-topic", data)

	// Read the message from the client
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	var wsMsg WSMessage
	if err := json.Unmarshal(msg, &wsMsg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if wsMsg.Topic != "test-topic" {
		t.Errorf("expected topic test-topic, got %s", wsMsg.Topic)
	}
	if wsMsg.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestHub_NoMessageForUnsubscribedTopic(t *testing.T) {
	hub := NewHub()
	hub.Start()
	defer hub.Stop()

	// Client subscribes to "topic-a"
	s := httptest.NewServer(http.HandlerFunc(hub.ServeWS("topic-a")))
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Broadcast to a different topic
	data, _ := json.Marshal(map[string]string{"key": "value"})
	hub.broadcast("topic-b", data)

	// Client should not receive the message (use short deadline)
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Error("expected no message for unsubscribed topic, but got one")
	}
}

func TestHub_ClientDisconnect(t *testing.T) {
	hub := NewHub()
	hub.Start()
	defer hub.Stop()

	s := httptest.NewServer(http.HandlerFunc(hub.ServeWS("test-topic")))
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	before := len(hub.clients)
	hub.mu.RUnlock()
	if before != 1 {
		t.Fatalf("expected 1 client, got %d", before)
	}

	// Disconnect
	conn.Close()
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	after := len(hub.clients)
	hub.mu.RUnlock()
	if after != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", after)
	}
}
