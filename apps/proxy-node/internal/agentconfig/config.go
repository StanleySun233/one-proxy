package agentconfig

import "os"

type Config struct {
	ControlPlaneURL    string
	NodeBootstrapToken string
	NodeAccessToken    string
	EnrollmentSecret   string
	NodeID             string
	NodeName           string
	NodeMode           string
	NodeScopeKey       string
	NodeParentID       string
	NodePublicHost     string
	NodeJoinPassword   string
	ListenAddr         string
	HTTPSListenAddr    string
	HeartbeatInterval  string
	PolicyStatePath    string
	RuntimeConfigPath  string
	PublicCertProvider string
	LetsEncryptEmail   string
	LetsEncryptCacheDir string
}

func Load() Config {
	return Config{
		ControlPlaneURL:    envOrDefault("CONTROL_PLANE_URL", ""),
		NodeBootstrapToken: envOrDefault("NODE_BOOTSTRAP_TOKEN", ""),
		NodeAccessToken:    envOrDefault("NODE_ACCESS_TOKEN", ""),
		EnrollmentSecret:   envOrDefault("NODE_ENROLLMENT_SECRET", ""),
		NodeID:             envOrDefault("NODE_ID", ""),
		NodeName:           envOrDefault("NODE_NAME", ""),
		NodeMode:           envOrDefault("NODE_MODE", "relay"),
		NodeScopeKey:       envOrDefault("NODE_SCOPE_KEY", ""),
		NodeParentID:       envOrDefault("NODE_PARENT_ID", ""),
		NodePublicHost:     envOrDefault("NODE_PUBLIC_HOST", ""),
		NodeJoinPassword:   envOrDefault("NODE_JOIN_PASSWORD", ""),
		ListenAddr:         envOrDefault("NODE_LISTEN_ADDR", ":2888"),
		HTTPSListenAddr:    envOrDefault("NODE_HTTPS_LISTEN_ADDR", ":2889"),
		HeartbeatInterval:  envOrDefault("NODE_HEARTBEAT_INTERVAL", "30s"),
		PolicyStatePath:    envOrDefault("NODE_POLICY_STATE_PATH", "runtime/node-policy-state.json"),
		RuntimeConfigPath:  envOrDefault("NODE_RUNTIME_CONFIG_PATH", "runtime/node-runtime.json"),
		PublicCertProvider: envOrDefault("PUBLIC_CERT_PROVIDER", "lets_encrypt"),
		LetsEncryptEmail:   envOrDefault("LETSENCRYPT_EMAIL", ""),
		LetsEncryptCacheDir: envOrDefault("LETSENCRYPT_CACHE_DIR", "runtime/autocert"),
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
