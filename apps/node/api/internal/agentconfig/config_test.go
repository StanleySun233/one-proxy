package agentconfig

import "testing"

func TestLoadUsesParentURLAsPanelBootstrapEndpoint(t *testing.T) {
	t.Setenv("NODE_PARENT_URL", "http://panel:2886")

	cfg := Load()

	if cfg.ControlPlaneURL != "http://panel:2886" {
		t.Fatalf("ControlPlaneURL = %q", cfg.ControlPlaneURL)
	}
}

func TestLoadUsesParentURLAsNodeBootstrapEndpoint(t *testing.T) {
	t.Setenv("NODE_PARENT_URL", "http://parent:2988")

	cfg := Load()

	if cfg.ControlPlaneURL != "http://parent:2988" {
		t.Fatalf("ControlPlaneURL = %q", cfg.ControlPlaneURL)
	}
}

func TestLoadUsesNodeOperationalDefaults(t *testing.T) {
	cfg := Load()

	if cfg.NodeLogRetention != "72h" {
		t.Fatalf("NodeLogRetention = %q", cfg.NodeLogRetention)
	}
	if cfg.NodeResponseCacheTTL != "1h" {
		t.Fatalf("NodeResponseCacheTTL = %q", cfg.NodeResponseCacheTTL)
	}
	if cfg.NodeResponseCacheMemory != "512mb" {
		t.Fatalf("NodeResponseCacheMemory = %q", cfg.NodeResponseCacheMemory)
	}
	if cfg.NodeResponseCacheDisk != "2gb" {
		t.Fatalf("NodeResponseCacheDisk = %q", cfg.NodeResponseCacheDisk)
	}
}

func TestLoadNormalizesSchemeLessParentURL(t *testing.T) {
	t.Setenv("NODE_PARENT_URL", "103.214.172.211:2988")

	cfg := Load()

	if cfg.ControlPlaneURL != "http://103.214.172.211:2988" {
		t.Fatalf("ControlPlaneURL = %q", cfg.ControlPlaneURL)
	}
}
