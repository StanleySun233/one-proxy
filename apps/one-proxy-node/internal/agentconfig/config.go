package agentconfig

import "os"

type Config struct {
	ControlPlaneURL          string
	NodeBootstrapToken       string
	NodeAccessToken          string
	EnrollmentSecret         string
	NodeID                   string
	NodeName                 string
	NodeMode                 string
	NodeScopeKey             string
	NodeParentID             string
	NodePublicHost           string
	NodeJoinPassword         string
	NodeJoinPasswordProvided bool
	NodeParentTunnelURL      string
	NodeTunnelPath           string
	NodeTunnelHeartbeat      string
	ListenAddr               string
	HTTPSListenAddr          string
	HeartbeatInterval        string
	PolicyStatePath          string
	RuntimeConfigPath        string
	PublicCertProvider       string
	LetsEncryptEmail         string
	LetsEncryptCacheDir      string
}

func Load() Config {
	joinPassword, joinPasswordProvided := lookupEnvOrDefault("NODE_JOIN_PASSWORD", "password")
	return Config{
		ControlPlaneURL:          envOrDefault("CONTROL_PLANE_URL", ""),
		NodeBootstrapToken:       envOrDefault("NODE_BOOTSTRAP_TOKEN", ""),
		NodeAccessToken:          envOrDefault("NODE_ACCESS_TOKEN", ""),
		EnrollmentSecret:         envOrDefault("NODE_ENROLLMENT_SECRET", ""),
		NodeID:                   envOrDefault("NODE_ID", ""),
		NodeName:                 envOrDefault("NODE_NAME", ""),
		NodeMode:                 envOrDefault("NODE_MODE", "relay"),
		NodeScopeKey:             envOrDefault("NODE_SCOPE_KEY", ""),
		NodeParentID:             envOrDefault("NODE_PARENT_ID", ""),
		NodePublicHost:           envOrDefault("NODE_PUBLIC_HOST", ""),
		NodeJoinPassword:         joinPassword,
		NodeJoinPasswordProvided: joinPasswordProvided,
		NodeParentTunnelURL:      envOrDefault("NODE_PARENT_TUNNEL_URL", ""),
		NodeTunnelPath:           envOrDefault("NODE_TUNNEL_PATH", "/api/v1/node-tunnel/connect"),
		NodeTunnelHeartbeat:      envOrDefault("NODE_TUNNEL_HEARTBEAT_INTERVAL", "15s"),
		ListenAddr:               envOrDefault("NODE_LISTEN_ADDR", ":2888"),
		HTTPSListenAddr:          envOrDefault("NODE_HTTPS_LISTEN_ADDR", ":2889"),
		HeartbeatInterval:        envOrDefault("NODE_HEARTBEAT_INTERVAL", "30s"),
		PolicyStatePath:          envOrDefault("NODE_POLICY_STATE_PATH", "runtime/node-policy-state.json"),
		RuntimeConfigPath:        envOrDefault("NODE_RUNTIME_CONFIG_PATH", "runtime/node-runtime.json"),
		PublicCertProvider:       envOrDefault("PUBLIC_CERT_PROVIDER", "lets_encrypt"),
		LetsEncryptEmail:         envOrDefault("LETSENCRYPT_EMAIL", ""),
		LetsEncryptCacheDir:      envOrDefault("LETSENCRYPT_CACHE_DIR", "runtime/autocert"),
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func lookupEnvOrDefault(key string, fallback string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback, false
	}
	return value, true
}
