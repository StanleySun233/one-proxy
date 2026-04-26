package cert

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/acme/autocert"
)

type LetsEncryptManager struct {
	manager *autocert.Manager
}

func NewLetsEncryptManager(email string, cacheDir string, host string) (*LetsEncryptManager, error) {
	if err := os.MkdirAll(filepath.Clean(cacheDir), 0o755); err != nil {
		return nil, err
	}
	return &LetsEncryptManager{
		manager: &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Email:      email,
			Cache:      autocert.DirCache(cacheDir),
			HostPolicy: autocert.HostWhitelist(host),
		},
	}, nil
}

func (m *LetsEncryptManager) HTTPHandler(fallback http.Handler) http.Handler {
	return m.manager.HTTPHandler(fallback)
}

func (m *LetsEncryptManager) TLSConfig() *tls.Config {
	return m.manager.TLSConfig()
}
