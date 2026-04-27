package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr              string
	MySQLDSN              string
	JWTSigningKey         string
	AdminPassword         string
	BootstrapTokenTTL     string
	NodeCertTTL           string
	PublicCertRenewWindow string
	SchedulerInterval     string
	SessionTTL            string
	NodeHeartbeatTTL      string
	PublicCertProvider    string
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

const defaultEnvFilePath = "./data/.env"

func EnvFilePath() string {
	path := os.Getenv("ENV_FILE_PATH")
	if path == "" {
		return defaultEnvFilePath
	}
	return path
}

func IsUnconfigured() bool {
	_, err := os.Stat(EnvFilePath())
	return os.IsNotExist(err)
}

func LoadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		_ = os.Setenv(strings.TrimSpace(key), strings.TrimSpace(value))
	}
	return scanner.Err()
}

func Load() Config {
	return Config{
		HTTPAddr:              envOrDefault("HTTP_ADDR", ":2887"),
		MySQLDSN:              envOrDefault("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/one_proxy?charset=utf8mb4&parseTime=true&loc=UTC"),
		JWTSigningKey:         envOrDefault("JWT_SIGNING_KEY", "change-me"),
		AdminPassword:         os.Getenv("ADMIN_PASSWORD"),
		BootstrapTokenTTL:     envOrDefault("BOOTSTRAP_TOKEN_TTL", "15m"),
		NodeCertTTL:           envOrDefault("NODE_CERT_TTL", "720h"),
		PublicCertRenewWindow: envOrDefault("PUBLIC_CERT_RENEW_WINDOW", "168h"),
		SchedulerInterval:     envOrDefault("SCHEDULER_INTERVAL", "1m"),
		SessionTTL:            envOrDefault("SESSION_TTL", "12h"),
		NodeHeartbeatTTL:      envOrDefault("NODE_HEARTBEAT_TTL", "2m"),
		PublicCertProvider:    envOrDefault("PUBLIC_CERT_PROVIDER", "lets_encrypt"),
	}
}
