package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsWriteTimeout = 10 * time.Second
	wsPongTimeout  = 60 * time.Second
	wsPingInterval = 30 * time.Second
)

// allowedWSOrigins caches the parsed CORS_ALLOWED_ORIGINS for WebSocket origin checks.
var allowedWSOrigins []string

func init() {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" || raw == "*" {
		allowedWSOrigins = nil // nil means allow all
	} else {
		for _, o := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowedWSOrigins = append(allowedWSOrigins, trimmed)
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		if allowedWSOrigins == nil {
			return true // development mode: allow all
		}
		origin := r.Header.Get("Origin")
		for _, allowed := range allowedWSOrigins {
			if allowed == origin {
				return true
			}
		}
		slog.Warn("websocket origin rejected", "origin", origin)
		return false
	},
}

// HandleLegacyWS returns an HTTP handler that upgrades to WebSocket and
// subscribes the client to the Hub. The topic is read from the ?topic=
// query parameter, defaulting to "*" (all topics).
func HandleLegacyWS(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topic := r.URL.Query().Get("topic")
		if topic == "" {
			topic = "*"
		}
		hub.ServeWS(topic)(w, r)
	}
}
