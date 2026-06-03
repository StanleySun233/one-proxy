package agentconfig

import "testing"

func TestLoadUsesPanelControlPlaneForPublicNode(t *testing.T) {
	t.Setenv("CONTROL_PLANE_URL", "http://panel:2886")

	cfg := Load()

	if cfg.ControlPlaneURL != "http://panel:2886" {
		t.Fatalf("ControlPlaneURL = %q", cfg.ControlPlaneURL)
	}
}

func TestLoadUsesParentControlPlaneForChildNode(t *testing.T) {
	t.Setenv("CONTROL_PLANE_URL", "http://panel:2886")
	t.Setenv("NODE_PARENT_URL", "http://parent:2988")
	t.Setenv("NODE_PROXY_TOKEN_CONTROL_PLANE_URL", "http://panel:2886")

	cfg := Load()

	if cfg.ControlPlaneURL != "http://parent:2988" {
		t.Fatalf("ControlPlaneURL = %q", cfg.ControlPlaneURL)
	}
	if cfg.NodeProxyTokenControlPlaneURL != "http://parent:2988" {
		t.Fatalf("NodeProxyTokenControlPlaneURL = %q", cfg.NodeProxyTokenControlPlaneURL)
	}
	if cfg.NodeParentTunnelURL != "http://parent:2988" {
		t.Fatalf("NodeParentTunnelURL = %q", cfg.NodeParentTunnelURL)
	}
}
