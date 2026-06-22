package tunnel

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/runtime"
	"github.com/gorilla/websocket"
)

func (c *Controller) writeMessage(conn *websocket.Conn, message Message) error {
	c.writeMu.Lock()
	err := conn.WriteJSON(message)
	c.writeMu.Unlock()
	return err
}

func (c *Controller) websocketURL(current runtime.Binding, parentNodeID string) (string, error) {
	if current.ControlPlaneURL == "" {
		return "", errors.New("missing_parent_tunnel_url")
	}
	base, err := url.Parse(current.ControlPlaneURL)
	if err != nil {
		return "", err
	}
	switch base.Scheme {
	case "https":
		base.Scheme = "wss"
	default:
		base.Scheme = "ws"
	}
	base.Path = c.tunnelPath
	query := base.Query()
	query.Set("parentNodeId", parentNodeID)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (c *Controller) report(current runtime.Binding, status string, lastHeartbeatAt string) {
	if c.manager != nil {
		listenerValue := domain.ListenerStatusDown
		if status == domain.TransportStatusConnected || status == domain.TransportStatusAvailable {
			listenerValue = domain.ListenerStatusUp
		}
		c.manager.SetListenerStatus("transport:"+domain.TransportTypeReverseWSParent, listenerValue)
	}
	client := controlplane.New(current.ControlPlaneURL, current.NodeAccessToken)
	address, err := c.websocketURL(current, current.NodeParentID)
	if err != nil {
		return
	}
	_, _ = client.UpsertTransport(domain.UpsertNodeTransportInput{
		TransportType:   domain.TransportTypeReverseWSParent,
		Direction:       "outbound",
		Address:         address,
		Status:          status,
		ParentNodeID:    current.NodeParentID,
		ConnectedAt:     lastHeartbeatAt,
		LastHeartbeatAt: lastHeartbeatAt,
		LatencyMs:       0,
		Details:         map[string]string{"source": "parent_tunnel"},
	})
}

func strconvPort(port int) string {
	if port <= 0 {
		return "0"
	}
	return strconv.Itoa(port)
}
