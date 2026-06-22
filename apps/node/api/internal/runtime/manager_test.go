package runtime

import (
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
)

func TestNextTick(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 4, 500*1000*1000, time.UTC)
	got := nextTick(now, 10*time.Second)
	want := time.Date(2026, 6, 14, 12, 0, 10, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("nextTick = %s, want %s", got, want)
	}
}

func TestManagerSetListenerStatusUpdatesHealthSnapshot(t *testing.T) {
	manager := New("", policystore.New(""), time.Second, map[string]string{"runtime": domain.ListenerStatusUp}, nil, false, "", false)
	manager.SetListenerStatus("transport:reverse_ws_parent", domain.ListenerStatusDown)

	listenerStatus, _ := manager.healthStatus()
	if listenerStatus["transport:reverse_ws_parent"] != domain.ListenerStatusDown {
		t.Fatalf("listener status = %q, want %q", listenerStatus["transport:reverse_ws_parent"], domain.ListenerStatusDown)
	}
}
