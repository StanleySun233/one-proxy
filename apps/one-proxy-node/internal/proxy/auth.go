package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultTokenCacheTTL = 24 * time.Hour
	reverseQueryTokenKey = "proxy_token"
	reverseCookieName    = "one_proxy_token"
)

type TokenValidator interface {
	ValidateProxyToken(ctx context.Context, tokenHash string) (TokenValidation, error)
}

type TokenValidation struct {
	Valid     bool
	ExpiresAt time.Time
	CacheTTL  time.Duration
}

type cachedTokenValidation struct {
	valid     bool
	expiresAt time.Time
}

type TokenAuthorizer struct {
	auth  AuthConfig
	cache *tokenCache
}

func NewTokenAuthorizer(auth AuthConfig) *TokenAuthorizer {
	return &TokenAuthorizer{
		auth:  auth,
		cache: &tokenCache{items: map[string]cachedTokenValidation{}},
	}
}

func (a *TokenAuthorizer) Validate(ctx context.Context, token string) bool {
	if a == nil || a.auth.Validator == nil {
		return true
	}
	if token == "" {
		return false
	}
	hash := proxyTokenHash(token)
	now := time.Now().UTC()
	if validation, ok := a.cache.get(hash, now); ok {
		return validation.valid
	}
	result, err := a.auth.Validator.ValidateProxyToken(ctx, hash)
	if err != nil {
		return false
	}
	ttl := result.CacheTTL
	if ttl <= 0 {
		ttl = a.auth.CacheTTL
	}
	if ttl <= 0 {
		ttl = defaultTokenCacheTTL
	}
	expiresAt := now.Add(ttl)
	if !result.ExpiresAt.IsZero() && result.ExpiresAt.Before(expiresAt) {
		expiresAt = result.ExpiresAt
	}
	a.cache.set(hash, cachedTokenValidation{
		valid:     result.Valid,
		expiresAt: expiresAt,
	})
	return result.Valid && now.Before(expiresAt)
}

func (s *Server) authorizeReverse(w http.ResponseWriter, req *http.Request) bool {
	token, source := reverseToken(req)
	if s.auth.Validator == nil {
		return true
	}
	if token == "" || !s.validateToken(req.Context(), token) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="one-proxy"`)
		http.Error(w, "reverse_auth_required", http.StatusUnauthorized)
		return false
	}
	if source == reverseTokenSourceQuery {
		s.setReverseTokenCookie(w, req, token)
	}
	return true
}

func (s *Server) authorizeForward(w http.ResponseWriter, req *http.Request) bool {
	token := forwardToken(req.Header.Get("Proxy-Authorization"))
	if s.auth.Validator == nil {
		return true
	}
	if token == "" || !s.validateToken(req.Context(), token) {
		w.Header().Set("Proxy-Authenticate", `Basic realm="one-proxy"`)
		http.Error(w, "proxy_auth_required", http.StatusProxyAuthRequired)
		return false
	}
	return true
}

func (s *Server) validateToken(ctx context.Context, token string) bool {
	return s.authorizer.Validate(ctx, token)
}

func (s *Server) setReverseTokenCookie(w http.ResponseWriter, req *http.Request, token string) {
	ttl := s.auth.CacheTTL
	if ttl <= 0 {
		ttl = defaultTokenCacheTTL
	}
	http.SetCookie(w, &http.Cookie{
		Name:     reverseCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   req.TLS != nil,
	})
}

func proxyTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type reverseTokenSource int

const (
	reverseTokenSourceNone reverseTokenSource = iota
	reverseTokenSourceHeader
	reverseTokenSourceQuery
	reverseTokenSourceCookie
)

func reverseToken(req *http.Request) (string, reverseTokenSource) {
	if token := bearerToken(req.Header.Get("Authorization")); token != "" {
		return token, reverseTokenSourceHeader
	}
	if req.URL != nil {
		if token := strings.TrimSpace(req.URL.Query().Get(reverseQueryTokenKey)); token != "" {
			return token, reverseTokenSourceQuery
		}
	}
	if cookie, err := req.Cookie(reverseCookieName); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token, reverseTokenSourceCookie
		}
	}
	return "", reverseTokenSourceNone
}

func forwardToken(header string) string {
	if token := bearerToken(header); token != "" {
		return token
	}
	const prefix = "Basic "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return ""
	}
	user, pass, ok := strings.Cut(string(payload), ":")
	if !ok || user != "token" {
		return ""
	}
	return strings.TrimSpace(pass)
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

type tokenCache struct {
	mu    sync.Mutex
	items map[string]cachedTokenValidation
}

func (c *tokenCache) get(tokenHash string, now time.Time) (cachedTokenValidation, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[tokenHash]
	if !ok {
		return cachedTokenValidation{}, false
	}
	if !item.expiresAt.IsZero() && !now.Before(item.expiresAt) {
		delete(c.items, tokenHash)
		return cachedTokenValidation{}, false
	}
	return item, true
}

func (c *tokenCache) set(tokenHash string, validation cachedTokenValidation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[tokenHash] = validation
}
