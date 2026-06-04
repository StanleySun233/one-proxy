package proxycommand

import (
	"fmt"
	"os"
	"strings"
)

func ReadToken(envName, filePath string) (string, error) {
	if filePath != "" {
		return readTokenFile(filePath)
	}
	if envName == "" {
		envName = "ONEPROXY_PROXY_TOKEN"
	}
	token := strings.TrimSpace(os.Getenv(envName))
	if token == "" {
		return "", fmt.Errorf("missing proxy token: set %s or pass --token-file", envName)
	}
	return token, nil
}

func readTokenFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(content))
	if token == "" {
		return "", fmt.Errorf("empty proxy token file: %s", path)
	}
	return token, nil
}
