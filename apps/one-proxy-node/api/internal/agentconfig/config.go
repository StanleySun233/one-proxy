package agentconfig

import "os"

type Config struct {
	ControlPlaneURL           string
	NodeBootstrapToken        string
	NodeAccessToken           string
	EnrollmentSecret          string
	NodeID                    string
	NodeName                  string
	NodeMode                  string
	NodeScopeKey              string
	NodeParentID              string
	NodePublicHost            string
	NodeJoinPassword          string
	NodeJoinPasswordProvided  bool
	NodeReverseTargetURL      string
	NodeProxyTokenCacheTTL    string
	NodeTunnelPath            string
	NodeTunnelHeartbeat       string
	NodeDirectListenAddr      string
	NodeDirectSTUNServers     string
	NodeDirectRefreshInterval string
	ListenAddr                string
	HTTPSListenAddr           string
	TCPAccessListenAddr       string
	UDPAccessListenAddr       string
	HeartbeatInterval         string
	PolicyStatePath           string
	RuntimeConfigPath         string
	PublicCertProvider        string
	LetsEncryptEmail          string
	LetsEncryptCacheDir       string
	NodeConsoleWebRoot        string
}

func Load() Config {
	joinPassword, joinPasswordProvided := lookupEnvOrDefault("NODE_JOIN_PASSWORD", "password")
	parentID := envOrDefault("NODE_PARENT_ID", "")
	parentURL := envOrDefault("NODE_PARENT_URL", "")
	return Config{
		ControlPlaneURL:           parentURL,
		NodeBootstrapToken:        envOrDefault("NODE_BOOTSTRAP_TOKEN", ""),
		NodeAccessToken:           envOrDefault("NODE_ACCESS_TOKEN", ""),
		EnrollmentSecret:          envOrDefault("NODE_ENROLLMENT_SECRET", ""),
		NodeID:                    envOrDefault("NODE_ID", ""),
		NodeName:                  envOrDefault("NODE_NAME", ""),
		NodeMode:                  envOrDefault("NODE_MODE", ""),
		NodeScopeKey:              envOrDefault("NODE_SCOPE_KEY", ""),
		NodeParentID:              parentID,
		NodePublicHost:            envOrDefault("NODE_PUBLIC_HOST", ""),
		NodeJoinPassword:          joinPassword,
		NodeJoinPasswordProvided:  joinPasswordProvided,
		NodeReverseTargetURL:      envOrDefault("NODE_REVERSE_TARGET_URL", ""),
		NodeProxyTokenCacheTTL:    envOrDefault("NODE_PROXY_TOKEN_CACHE_TTL", "24h"),
		NodeTunnelPath:            envOrDefault("NODE_TUNNEL_PATH", "/api/node-tunnel/connect"),
		NodeTunnelHeartbeat:       envOrDefault("NODE_TUNNEL_HEARTBEAT_INTERVAL", "15s"),
		NodeDirectListenAddr:      envOrDefault("NODE_DIRECT_LISTEN_ADDR", ""),
		NodeDirectSTUNServers:     envOrDefault("NODE_DIRECT_STUN_SERVERS", "stun.cloudflare.com:3478"),
		NodeDirectRefreshInterval: envOrDefault("NODE_DIRECT_REFRESH_INTERVAL", "30s"),
		ListenAddr:                envOrDefault("NODE_LISTEN_ADDR", ":2988"),
		HTTPSListenAddr:           envOrDefault("NODE_HTTPS_LISTEN_ADDR", ":2989"),
		TCPAccessListenAddr:       envOrDefault("NODE_TCP_ACCESS_LISTEN_ADDR", ""),
		UDPAccessListenAddr:       envOrDefault("NODE_UDP_ACCESS_LISTEN_ADDR", ""),
		HeartbeatInterval:         envOrDefault("NODE_HEARTBEAT_INTERVAL", "30s"),
		PolicyStatePath:           envOrDefault("NODE_POLICY_STATE_PATH", "runtime/node-policy-state.json"),
		RuntimeConfigPath:         envOrDefault("NODE_RUNTIME_CONFIG_PATH", "runtime/node-runtime.json"),
		PublicCertProvider:        envOrDefault("PUBLIC_CERT_PROVIDER", "lets_encrypt"),
		LetsEncryptEmail:          envOrDefault("LETSENCRYPT_EMAIL", ""),
		LetsEncryptCacheDir:       envOrDefault("LETSENCRYPT_CACHE_DIR", "runtime/autocert"),
		NodeConsoleWebRoot:        envOrDefault("NODE_CONSOLE_WEB_ROOT", "web"),
	}
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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
