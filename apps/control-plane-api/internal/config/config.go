package config

import "os"

type Config struct {
	HTTPAddr             string
	SQLitePath           string
	JWTSigningKey        string
	BootstrapTokenTTL    string
	NodeCertTTL          string
	PublicCertRenewWindow string
	SchedulerInterval    string
	SessionTTL           string
	NodeHeartbeatTTL     string
	PublicCertProvider   string
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func Load() Config {
	return Config{
		HTTPAddr:              envOrDefault("HTTP_ADDR", ":2887"),
		SQLitePath:            envOrDefault("SQLITE_PATH", "runtime/control-plane.db"),
		JWTSigningKey:         envOrDefault("JWT_SIGNING_KEY", "change-me"),
		BootstrapTokenTTL:     envOrDefault("BOOTSTRAP_TOKEN_TTL", "15m"),
		NodeCertTTL:           envOrDefault("NODE_CERT_TTL", "720h"),
		PublicCertRenewWindow: envOrDefault("PUBLIC_CERT_RENEW_WINDOW", "168h"),
		SchedulerInterval:     envOrDefault("SCHEDULER_INTERVAL", "1m"),
		SessionTTL:            envOrDefault("SESSION_TTL", "12h"),
		NodeHeartbeatTTL:      envOrDefault("NODE_HEARTBEAT_TTL", "2m"),
		PublicCertProvider:    envOrDefault("PUBLIC_CERT_PROVIDER", "lets_encrypt"),
	}
}
