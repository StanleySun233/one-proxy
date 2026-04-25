package config

import "os"

type Config struct {
	HTTPAddr             string
	SQLitePath           string
	JWTSigningKey        string
	BootstrapTokenTTL    string
	NodeCertTTL          string
	PublicCertRenewWindow string
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
		HTTPAddr:              envOrDefault("HTTP_ADDR", ":8080"),
		SQLitePath:            envOrDefault("SQLITE_PATH", "runtime/control-plane.db"),
		JWTSigningKey:         envOrDefault("JWT_SIGNING_KEY", "change-me"),
		BootstrapTokenTTL:     envOrDefault("BOOTSTRAP_TOKEN_TTL", "15m"),
		NodeCertTTL:           envOrDefault("NODE_CERT_TTL", "720h"),
		PublicCertRenewWindow: envOrDefault("PUBLIC_CERT_RENEW_WINDOW", "168h"),
	}
}
