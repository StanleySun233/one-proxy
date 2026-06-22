package tunnel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRegistryAddClosesPreviousSession(t *testing.T) {
	registry := NewRegistry()
	previous := registry.Add("node-1", nil)
	registry.Add("node-1", nil)

	select {
	case <-previous.done:
	case <-time.After(time.Second):
		t.Fatal("previous session was not closed")
	}
}

func TestRegistryRemoveIsIdempotent(t *testing.T) {
	registry := NewRegistry()
	session := registry.Add("node-1", nil)

	registry.Remove("node-1", session)
	registry.Remove("node-1", session)
}

func TestRegistryWaitForSessionReturnsNewSession(t *testing.T) {
	registry := NewRegistry()
	result := make(chan *childSession, 1)
	errCh := make(chan error, 1)
	go func() {
		session, err := registry.waitForSession("node-1", time.Second)
		if err != nil {
			errCh <- err
			return
		}
		result <- session
	}()

	created := registry.Add("node-1", nil)

	select {
	case err := <-errCh:
		t.Fatalf("waitForSession error = %v", err)
	case got := <-result:
		if got != created {
			t.Fatal("waitForSession returned a different session")
		}
	case <-time.After(time.Second):
		t.Fatal("waitForSession did not return")
	}
}

func TestRegistryOpenStreamWaitsForReconnect(t *testing.T) {
	registry := NewRegistry()
	result := make(chan error, 1)
	go func() {
		conn, err := registry.OpenStream("node-1", nil, "target.local", 443)
		if err == nil {
			result <- nil
			_ = conn.Close()
			return
		}
		result <- err
	}()

	clientConn := testWebsocketClient(t, func(conn *websocket.Conn) {
		var message Message
		if err := conn.ReadJSON(&message); err != nil {
			return
		}
		_ = conn.WriteJSON(Message{Type: "open_ack", StreamID: message.StreamID, Status: "connected"})
	})
	runSessionReader(registry.Add("node-1", clientConn), clientConn)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("OpenStream error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("OpenStream did not use reconnected session")
	}
}

func TestRegistryOpenStreamRetriesAfterClosedSession(t *testing.T) {
	registry := NewRegistry()
	oldConn := testWebsocketClient(t, func(conn *websocket.Conn) {
		var message Message
		_ = conn.ReadJSON(&message)
	})
	oldSession := registry.Add("node-1", oldConn)
	result := make(chan error, 1)
	go func() {
		conn, err := registry.OpenStream("node-1", nil, "target.local", 443)
		if err == nil {
			result <- nil
			_ = conn.Close()
			return
		}
		result <- err
	}()

	time.Sleep(20 * time.Millisecond)
	registry.Remove("node-1", oldSession)
	clientConn := testWebsocketClient(t, func(conn *websocket.Conn) {
		var message Message
		if err := conn.ReadJSON(&message); err != nil {
			return
		}
		_ = conn.WriteJSON(Message{Type: "open_ack", StreamID: message.StreamID, Status: "connected"})
	})
	runSessionReader(registry.Add("node-1", clientConn), clientConn)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("OpenStream error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("OpenStream did not retry with reconnected session")
	}
}

func testWebsocketClient(t *testing.T, handler func(*websocket.Conn)) *websocket.Conn {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error = %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

func runSessionReader(session *childSession, conn *websocket.Conn) {
	go func() {
		for {
			var message Message
			if err := conn.ReadJSON(&message); err != nil {
				return
			}
			session.handleMessage(message)
		}
	}()
}
