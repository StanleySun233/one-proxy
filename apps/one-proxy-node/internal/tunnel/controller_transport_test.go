package tunnel

import (
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/runtime"
)

func TestControllerWebsocketURLFallsBackToRuntimeControlPlane(t *testing.T) {
	controller := NewController(nil, nil, "", "/api/v1/node-tunnel/connect", 15*time.Second)
	current := runtime.Binding{
		ControlPlaneURL: "http://parent.example:2988",
		NodeParentID:    "1",
	}

	got, err := controller.websocketURL(current, current.NodeParentID)
	if err != nil {
		t.Fatalf("websocketURL error = %v", err)
	}
	want := "ws://parent.example:2988/api/v1/node-tunnel/connect?parentNodeId=1"
	if got != want {
		t.Fatalf("websocketURL = %q, want %q", got, want)
	}
}

func TestControllerWebsocketURLPrefersExplicitParentTunnelURL(t *testing.T) {
	controller := NewController(nil, nil, "https://public.example:2988", "/api/v1/node-tunnel/connect", 15*time.Second)
	current := runtime.Binding{
		ControlPlaneURL: "http://parent.example:2988",
		NodeParentID:    "1",
	}

	got, err := controller.websocketURL(current, current.NodeParentID)
	if err != nil {
		t.Fatalf("websocketURL error = %v", err)
	}
	want := "wss://public.example:2988/api/v1/node-tunnel/connect?parentNodeId=1"
	if got != want {
		t.Fatalf("websocketURL = %q, want %q", got, want)
	}
}
