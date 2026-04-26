package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/service"
)

type APIResponse[T any] struct {
	Code    int  `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type contextKey string

const accountContextKey contextKey = "account"
const nodeContextKey contextKey = "node"

func writeSuccess[T any](w http.ResponseWriter, status int, data T) {
	writeEnvelope(w, status, APIResponse[T]{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeEnvelope(w, status, APIResponse[any]{
		Code:    status,
		Message: message,
	})
}

func writeServiceError(w http.ResponseWriter, req *http.Request, err error, fallback string) {
	if err == nil {
		log.Printf("http service error method=%s path=%s status=%d code=%s err=nil", req.Method, req.URL.Path, http.StatusInternalServerError, fallback)
		writeError(w, http.StatusInternalServerError, fallback)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("http service error method=%s path=%s status=%d code=%s err=%v", req.Method, req.URL.Path, http.StatusNotFound, "resource_not_found", err)
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if serviceErr, ok := err.(*service.Error); ok {
		log.Printf("http service error method=%s path=%s status=%d code=%s err=%v", req.Method, req.URL.Path, serviceErr.Status, serviceErr.Code, err)
		writeError(w, serviceErr.Status, serviceErr.Message)
		return
	}
	log.Printf("http service error method=%s path=%s status=%d code=%s err=%v", req.Method, req.URL.Path, http.StatusInternalServerError, fallback, err)
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeEnvelope[T any](w http.ResponseWriter, status int, payload APIResponse[T]) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeMethodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
}

func bearerToken(req *http.Request) string {
	header := strings.TrimSpace(req.Header.Get("Authorization"))
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (r *Router) requireAccount(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		token := bearerToken(req)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing_bearer_token")
			return
		}
		account, ok := r.service.AuthenticateAccessToken(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_access_token")
			return
		}
		if account.MustRotatePassword {
			if !(req.Method == http.MethodPatch && req.URL.Path == "/api/v1/accounts/"+account.ID) {
				writeError(w, http.StatusForbidden, "password_rotation_required")
				return
			}
		}
		ctx := context.WithValue(req.Context(), accountContextKey, account)
		next(w, req.WithContext(ctx))
	}
}

func accountFromContext(ctx context.Context) (domain.Account, bool) {
	account, ok := ctx.Value(accountContextKey).(domain.Account)
	return account, ok
}

func (r *Router) requireNode(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		token := bearerToken(req)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing_bearer_token")
			return
		}
		nodeID, ok := r.service.AuthenticateNodeToken(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_node_token")
			return
		}
		ctx := context.WithValue(req.Context(), nodeContextKey, nodeID)
		next(w, req.WithContext(ctx))
	}
}

func nodeIDFromContext(ctx context.Context) (string, bool) {
	nodeID, ok := ctx.Value(nodeContextKey).(string)
	return nodeID, ok
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		startedAt := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("http panic method=%s path=%s err=%v\n%s", req.Method, req.URL.Path, recovered, debug.Stack())
				writeError(sw, http.StatusInternalServerError, "internal_server_error")
			}
			if sw.status >= http.StatusBadRequest {
				log.Printf("http request method=%s path=%s status=%d duration=%s", req.Method, req.URL.Path, sw.status, time.Since(startedAt))
			}
		}()
		next.ServeHTTP(sw, req)
	})
}
