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
