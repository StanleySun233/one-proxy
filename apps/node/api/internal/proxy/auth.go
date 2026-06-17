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
	reverseHeaderName    = "X-One-Proxy-Token"
	reverseQueryTokenKey = "one_proxy_auth"
	reverseCookieName    = "one_proxy_auth"
)

type TokenValidator interface {
	ValidateProxyToken(ctx context.Context, tokenHash string) (TokenValidation, error)
}

type TokenValidation struct {
	Valid           bool
	ExpiresAt       time.Time
	CacheTTL        time.Duration
	AllowLocalProxy bool
	TenantID        string
}

type cachedTokenValidation struct {
	valid           bool
	expiresAt       time.Time
	allowLocalProxy bool
	tenantID        string
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
	return a.Authorize(ctx, token).Valid
}

func (a *TokenAuthorizer) Authorize(ctx context.Context, token string) TokenValidation {
	if a == nil || a.auth.Validator == nil {
		return TokenValidation{}
	}
	if token == "" {
		return TokenValidation{}
	}
	hash := proxyTokenHash(token)
	now := time.Now().UTC()
	if validation, ok := a.cache.get(hash, now); ok {
		return TokenValidation{
			Valid:           validation.valid,
			ExpiresAt:       validation.expiresAt,
			AllowLocalProxy: validation.allowLocalProxy,
			TenantID:        validation.tenantID,
		}
	}
	result, err := a.auth.Validator.ValidateProxyToken(ctx, hash)
	if err != nil {
		return TokenValidation{}
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
	if !result.Valid || !now.Before(expiresAt) {
		return TokenValidation{}
	}
	a.cache.set(hash, cachedTokenValidation{
		valid:           result.Valid,
		expiresAt:       expiresAt,
		allowLocalProxy: result.AllowLocalProxy,
		tenantID:        result.TenantID,
	})
	return TokenValidation{
		Valid:           true,
		ExpiresAt:       expiresAt,
		AllowLocalProxy: result.AllowLocalProxy,
		TenantID:        result.TenantID,
	}
}

func (s *Server) authorizeReverse(w http.ResponseWriter, req *http.Request) (TokenValidation, bool) {
	token, source := reverseToken(req)
	validation := s.authorizer.Authorize(req.Context(), token)
	if token == "" || !validation.Valid {
		w.Header().Set("X-One-Proxy-Authenticate", "required")
		writeProxyError(w, req, proxyErrorReverseAuthRequired, http.StatusUnauthorized)
		return TokenValidation{}, false
	}
	if source == reverseTokenSourceQuery {
		s.setReverseTokenCookie(w, req, token)
	}
	return validation, true
}

func (s *Server) authorizeForward(w http.ResponseWriter, req *http.Request) bool {
	_, ok := s.authorizeForwardRequest(w, req)
	return ok
}

func (s *Server) authorizeForwardRequest(w http.ResponseWriter, req *http.Request) (TokenValidation, bool) {
	token := forwardToken(req.Header.Get("Proxy-Authorization"))
	validation := s.authorizer.Authorize(req.Context(), token)
	if token == "" || !validation.Valid {
		w.Header().Set("Proxy-Authenticate", `Basic realm="one-proxy"`)
		writeProxyError(w, req, proxyErrorProxyAuthRequired, http.StatusProxyAuthRequired)
		return TokenValidation{}, false
	}
	return validation, true
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
	if token := strings.TrimSpace(req.Header.Get(reverseHeaderName)); token != "" {
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
