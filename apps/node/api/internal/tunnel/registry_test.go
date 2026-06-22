package tunnel

import (
	"testing"
	"time"
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
