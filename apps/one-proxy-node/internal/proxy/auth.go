package proxy

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

func (s *Server) authorizeReverse(w http.ResponseWriter, req *http.Request) bool {
	if s.auth.ReverseUser == "" && s.auth.ReversePassword == "" {
		return true
	}
	if basicAuthMatches(req.Header.Get("Authorization"), s.auth.ReverseUser, s.auth.ReversePassword) {
		return true
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="one-proxy"`)
	http.Error(w, "reverse_auth_required", http.StatusUnauthorized)
	return false
}

func (s *Server) authorizeForward(w http.ResponseWriter, req *http.Request) bool {
	if s.auth.ForwardUser == "" && s.auth.ForwardPassword == "" {
		return true
	}
	if basicAuthMatches(req.Header.Get("Proxy-Authorization"), s.auth.ForwardUser, s.auth.ForwardPassword) {
		return true
	}
	w.Header().Set("Proxy-Authenticate", `Basic realm="one-proxy"`)
	http.Error(w, "proxy_auth_required", http.StatusProxyAuthRequired)
	return false
}

func basicAuthMatches(header string, username string, password string) bool {
	const prefix = "Basic "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return false
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(string(payload), ":")
	if !ok {
		return false
	}
	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1
	return userMatch && passMatch
}
