package server

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development; restrict in production.
		return true
	},
}

// HandleWebSocket upgrades an HTTP connection to WebSocket and manages the
// bidirectional message loop. This is a stub implementation.
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	slog.Info("websocket connected", "remote_addr", r.RemoteAddr)

	// Stub: read messages and echo back until the connection closes.
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("websocket read error", "error", err)
			}
			break
		}

		if err := conn.WriteMessage(msgType, msg); err != nil {
			slog.Error("websocket write error", "error", err)
			break
		}
	}
}
