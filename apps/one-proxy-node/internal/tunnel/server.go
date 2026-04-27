package tunnel

import (
	"log"
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/runtime"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func NewServer(manager *runtime.Manager, registry *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !manager.Bound() {
			http.Error(w, "control_plane_unbound", http.StatusServiceUnavailable)
			return
		}
		current := manager.Current()
		parentNodeID := req.URL.Query().Get("parentNodeId")
		if parentNodeID == "" || parentNodeID != current.NodeID {
			http.Error(w, "invalid_parent_node", http.StatusForbidden)
			return
		}
		token := bearerToken(req)
		if token == "" {
			http.Error(w, "missing_bearer_token", http.StatusUnauthorized)
			return
		}
		validator := controlplane.New(current.ControlPlaneURL, token)
		policy, err := validator.FetchPolicy()
		if err != nil || policy.NodeID == "" {
			http.Error(w, "invalid_child_node_token", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		session := registry.Add(policy.NodeID, conn)
		defer registry.Remove(policy.NodeID, session)
		_ = conn.SetReadDeadline(time.Now().Add(45 * time.Second))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(45 * time.Second))
		})
		for {
			var message Message
			if err := conn.ReadJSON(&message); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(45 * time.Second))
			if message.Type == "probe_response" {
				session.resolve(message)
				continue
			}
			if message.Type == "open_ack" || message.Type == "stream_data" || message.Type == "close_stream" {
				session.handleMessage(message)
				continue
			}
			if message.Type == "heartbeat" {
				session.writeMu.Lock()
				_ = conn.WriteJSON(Message{
					Type:      "heartbeat_ack",
					NodeID:    policy.NodeID,
					ParentID:  current.NodeID,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Status:    "connected",
				})
				session.writeMu.Unlock()
			}
			if message.Type == "register" {
				log.Printf("node tunnel child_connected childNodeID=%s parentNodeID=%s", policy.NodeID, current.NodeID)
				session.writeMu.Lock()
				_ = conn.WriteJSON(Message{
					Type:      "register_ack",
					NodeID:    policy.NodeID,
					ParentID:  current.NodeID,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Status:    "connected",
				})
				session.writeMu.Unlock()
			}
		}
	})
}

func bearerToken(req *http.Request) string {
	header := req.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return ""
	}
	return header[len(prefix):]
}
