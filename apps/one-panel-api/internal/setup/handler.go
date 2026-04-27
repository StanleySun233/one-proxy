package setup

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type SetupHandler struct {
	envFilePath  string
	transitionFn func() error
	mu           sync.Mutex
}

func NewSetupHandler(envFilePath string, transitionFn func() error) *SetupHandler {
	return &SetupHandler{
		envFilePath:  envFilePath,
		transitionFn: transitionFn,
	}
}

func (h *SetupHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.handleHealthz)
	mux.HandleFunc("/api/v1/setup/status", h.handleStatus)
	mux.HandleFunc("/api/v1/setup/test-connection", h.handleTestConnection)
	mux.HandleFunc("/api/v1/setup/generate-key", h.handleGenerateKey)
	mux.HandleFunc("/api/v1/setup/init", h.handleInit)
}

// --- Response helpers ---

type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func writeSuccess[T any](w http.ResponseWriter, status int, data T) {
	writeEnvelope(w, status, apiResponse[T]{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeEnvelope(w, status, apiResponse[any]{
		Code:    status,
		Message: message,
	})
}

func writeEnvelope[T any](w http.ResponseWriter, status int, payload apiResponse[T]) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeMethodNotAllowed(w http.ResponseWriter, allowedMethod string) {
	w.Header().Set("Allow", allowedMethod)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
}

// --- Request types ---

type testConnectionRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type initRequest struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Database       string `json:"database"`
	JWTSigningKey  string `json:"jwtSigningKey"`
	AdminPassword  string `json:"adminPassword"`
	NeedInitialize bool   `json:"needInitialize"`
}

// --- Handlers ---

func (h *SetupHandler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, map[string]string{
		"status": "ok",
		"mode":   "setup",
	})
}

func (h *SetupHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	_, err := os.Stat(h.envFilePath)
	writeSuccess(w, http.StatusOK, map[string]bool{
		"configured": err == nil,
	})
}

func (h *SetupHandler) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	var req testConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_body")
		return
	}

	dsn := buildDSN(req.User, req.Password, req.Host, req.Port, req.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		writeSuccess(w, http.StatusOK, map[string]any{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	defer db.Close()

	db.SetConnMaxLifetime(5 * time.Second)
	if err := db.Ping(); err != nil {
		writeSuccess(w, http.StatusOK, map[string]any{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	writeSuccess(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "connection_ok",
	})
}

func (h *SetupHandler) handleGenerateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		writeError(w, http.StatusInternalServerError, "generate_key_failed")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]string{
		"key": hex.EncodeToString(key),
	})
}

func (h *SetupHandler) handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	var req initRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_body")
		return
	}
	if req.AdminPassword == "" {
		writeError(w, http.StatusBadRequest, "admin_password_required")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	dsn := buildDSN(req.User, req.Password, req.Host, req.Port, req.Database)

	if req.NeedInitialize {
		initDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=true&loc=UTC",
			req.User, req.Password, req.Host, req.Port)
		initDB, err := sql.Open("mysql", initDSN)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := initDB.Exec("CREATE DATABASE IF NOT EXISTS `" + req.Database + "`"); err != nil {
			initDB.Close()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		initDB.Close()
	}

	// Validate the full DSN connection before writing .env
	testDB, err := sql.Open("mysql", dsn)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	testDB.SetConnMaxLifetime(5 * time.Second)
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	testDB.Close()

	envContent := fmt.Sprintf("MYSQL_DSN=%s\nJWT_SIGNING_KEY=%s\n", dsn, req.JWTSigningKey)
	if req.AdminPassword != "" {
		envContent += fmt.Sprintf("ADMIN_PASSWORD=%s\n", req.AdminPassword)
	}
	if err := os.WriteFile(h.envFilePath, []byte(envContent), 0600); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.transitionFn(); err != nil {
		os.Remove(h.envFilePath)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "initialized",
	})
}

// --- Helpers ---

func buildDSN(user, password, host string, port int, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		user, password, host, port, database)
}
