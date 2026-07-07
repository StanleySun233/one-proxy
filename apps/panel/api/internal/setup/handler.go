package setup

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

const setupDBPingTimeout = 5 * time.Second

type SetupHandler struct {
	envFilePath      string
	transitionFn     func() error
	configuredFn     func() bool
	allowExistingEnv bool
	mu               sync.Mutex
}

func NewSetupHandler(envFilePath string, transitionFn func() error) *SetupHandler {
	return &SetupHandler{
		envFilePath:  envFilePath,
		transitionFn: transitionFn,
	}
}

func (h *SetupHandler) WithConfiguredFunc(configuredFn func() bool) *SetupHandler {
	h.configuredFn = configuredFn
	return h
}

func (h *SetupHandler) WithExistingEnvAllowed() *SetupHandler {
	h.allowExistingEnv = true
	return h
}

func (h *SetupHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.handleHealthz)
	mux.HandleFunc("/api/setup/status", h.handleStatus)
	mux.HandleFunc("/api/setup/test", h.handleTestConnection)
	mux.HandleFunc("/api/setup/key", h.handleGenerateKey)
	mux.HandleFunc("/api/setup/init", h.handleInit)
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
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Database       string `json:"database"`
	NeedInitialize bool   `json:"needInitialize"`
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
	if h.configuredFn != nil {
		writeSuccess(w, http.StatusOK, map[string]bool{
			"configured": h.configuredFn(),
		})
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
	if err := validateSetupDBRequest(req.Host, req.Port, req.User, req.Database); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	database := req.Database
	if req.NeedInitialize {
		database = ""
	}
	dsn := buildDSN(req.User, req.Password, req.Host, req.Port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("setup test sql open failed: %v", err)
		writeConnectionTestFailure(w)
		return
	}
	defer db.Close()

	configureSetupDB(db)
	ctx, cancel := context.WithTimeout(r.Context(), setupDBPingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Printf("setup test ping failed: %v", err)
		writeConnectionTestFailure(w)
		return
	}

	var tableCount int
	if req.Database != "" {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", req.Database).Scan(&tableCount); err != nil {
			log.Printf("setup test table count failed: %v", err)
			writeConnectionTestFailure(w)
			return
		}
	}

	writeSuccess(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "connection_ok",
		"exists":  tableCount > 0,
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
	if err := validateSetupDBRequest(req.Host, req.Port, req.User, req.Database); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := os.Stat(h.envFilePath)
	if err == nil && !h.allowExistingEnv {
		writeError(w, http.StatusConflict, "setup_already_configured")
		return
	} else if err != nil && !os.IsNotExist(err) {
		log.Printf("setup env stat failed: %v", err)
		writeError(w, http.StatusInternalServerError, "setup_status_failed")
		return
	}

	dsn := buildDSN(req.User, req.Password, req.Host, req.Port, req.Database)

	if req.NeedInitialize {
		initDSN := buildDSN(req.User, req.Password, req.Host, req.Port, "")
		initDB, err := sql.Open("mysql", initDSN)
		if err != nil {
			log.Printf("setup init sql open failed: %v", err)
			writeError(w, http.StatusInternalServerError, "database_connection_failed")
			return
		}
		configureSetupDB(initDB)
		ctx, cancel := context.WithTimeout(r.Context(), setupDBPingTimeout)
		_, err = initDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+req.Database+"`")
		cancel()
		if err != nil {
			initDB.Close()
			log.Printf("setup create database failed: %v", err)
			writeError(w, http.StatusInternalServerError, "database_initialization_failed")
			return
		}
		initDB.Close()
	}

	// Validate the full DSN connection before writing .env
	testDB, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("setup init test sql open failed: %v", err)
		writeError(w, http.StatusInternalServerError, "database_connection_failed")
		return
	}
	configureSetupDB(testDB)
	ctx, cancel := context.WithTimeout(r.Context(), setupDBPingTimeout)
	err = testDB.PingContext(ctx)
	cancel()
	if err != nil {
		testDB.Close()
		log.Printf("setup init test ping failed: %v", err)
		writeError(w, http.StatusInternalServerError, "database_connection_failed")
		return
	}
	testDB.Close()

	envContent := fmt.Sprintf("MYSQL_DSN=%s\nJWT_SIGNING_KEY=%s\n", dsn, req.JWTSigningKey)
	if req.AdminPassword != "" {
		envContent += fmt.Sprintf("ADMIN_PASSWORD=%s\n", req.AdminPassword)
	}
	if err := os.WriteFile(h.envFilePath, []byte(envContent), 0600); err != nil {
		log.Printf("setup env write failed: %v", err)
		writeError(w, http.StatusInternalServerError, "setup_write_failed")
		return
	}

	if err := h.transitionFn(); err != nil {
		os.Remove(h.envFilePath)
		log.Printf("setup transition failed: %v", err)
		writeError(w, http.StatusInternalServerError, "setup_transition_failed")
		return
	}

	writeSuccess(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "initialized",
	})
}

// --- Helpers ---

func buildDSN(user, password, host string, port int, database string) string {
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	cfg.DBName = database
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	return cfg.FormatDSN()
}

func configureSetupDB(db *sql.DB) {
	db.SetConnMaxLifetime(setupDBPingTimeout)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
}

func validateSetupDBRequest(host string, port int, user string, database string) error {
	if !validSetupHost(host) || port < 1 || port > 65535 || user == "" {
		return fmt.Errorf("invalid_database_connection")
	}
	if !validDatabaseName(database) {
		return fmt.Errorf("invalid_database_name")
	}
	return nil
}

func validSetupHost(host string) bool {
	if host == "" || strings.TrimSpace(host) != host {
		return false
	}
	for _, char := range host {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '.' || char == '_' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func validDatabaseName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' {
			continue
		}
		return false
	}
	return true
}

func writeConnectionTestFailure(w http.ResponseWriter) {
	writeSuccess(w, http.StatusOK, map[string]any{
		"success": false,
		"message": "connection_failed",
	})
}
