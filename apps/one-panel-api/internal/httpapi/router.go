package httpapi

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/network"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/service"

	_ "github.com/go-sql-driver/mysql"
)

type Router struct {
	mux     *http.ServeMux
	service *service.ControlPlane
}

func NewRouter(cfg HTTPConfig, service *service.ControlPlane) http.Handler {
	router := &Router{
		mux:     http.NewServeMux(),
		service: service,
	}
	router.routes(cfg)
	return withObservability(router.mux)
}

type HTTPConfig struct {
	HTTPAddr    string
	DBBackend   string
	EnvFilePath string
}

func (r *Router) routes(cfg HTTPConfig) {
	r.mux.HandleFunc("/api/v1/setup/status", func(w http.ResponseWriter, _ *http.Request) {
		configured := r.service.IsInitialized()
		writeSuccess(w, http.StatusOK, map[string]any{"configured": configured})
	})

	r.mux.HandleFunc("/api/v1/setup/test-connection", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		var body struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
			Database string `json:"database"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_body")
			return
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
			body.User, body.Password, body.Host, body.Port, body.Database)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			writeSuccess(w, http.StatusOK, map[string]any{"success": false, "message": err.Error()})
			return
		}
		defer db.Close()
		db.SetConnMaxLifetime(5 * time.Second)
		if err := db.Ping(); err != nil {
			writeSuccess(w, http.StatusOK, map[string]any{"success": false, "message": err.Error()})
			return
		}
		var tableCount int
		_ = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", body.Database).Scan(&tableCount)
		writeSuccess(w, http.StatusOK, map[string]any{"success": true, "message": "connection_ok", "exists": tableCount > 0})
	})

	r.mux.HandleFunc("/api/v1/setup/generate-key", func(w http.ResponseWriter, req *http.Request) {
		key := make([]byte, 16)
		if _, err := rand.Read(key); err != nil {
			writeError(w, http.StatusInternalServerError, "generate_key_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]string{"key": hex.EncodeToString(key)})
	})

	r.mux.HandleFunc("/api/v1/setup/init", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		var body struct {
			Host           string `json:"host"`
			Port           int    `json:"port"`
			User           string `json:"user"`
			Password       string `json:"password"`
			Database       string `json:"database"`
			JWTSigningKey  string `json:"jwtSigningKey"`
			AdminPassword  string `json:"adminPassword"`
			NeedInitialize bool   `json:"needInitialize"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_body")
			return
		}
		if body.AdminPassword == "" {
			writeError(w, http.StatusBadRequest, "admin_password_required")
			return
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
			body.User, body.Password, body.Host, body.Port, body.Database)

		if body.NeedInitialize {
			initDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=true&loc=UTC",
				body.User, body.Password, body.Host, body.Port)
			initDB, err := sql.Open("mysql", initDSN)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if _, err := initDB.Exec("CREATE DATABASE IF NOT EXISTS `" + body.Database + "`"); err != nil {
				initDB.Close()
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			initDB.Close()
		}

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

		if cfg.EnvFilePath != "" {
			envContent := fmt.Sprintf("MYSQL_DSN=%s\nJWT_SIGNING_KEY=%s\nADMIN_PASSWORD=%s\n", dsn, body.JWTSigningKey, body.AdminPassword)
			if err := os.WriteFile(cfg.EnvFilePath, []byte(envContent), 0600); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			os.Setenv("ADMIN_PASSWORD", body.AdminPassword)
		}

		if err := r.service.ReinitializeStore(body.AdminPassword); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"success": true, "message": "initialized"})
	})

	r.mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]any{
			"status":    "ok",
			"httpAddr":  cfg.HTTPAddr,
			"dbBackend": cfg.DBBackend,
			"localIPs":  network.LocalIPs(),
		})
	})

	r.mux.HandleFunc("/api/v1/enums", r.handleEnums)
	r.mux.HandleFunc("/api/v1/auth/login", r.handleLogin)
	r.mux.HandleFunc("/api/v1/auth/refresh", r.handleRefresh)
	r.mux.HandleFunc("/api/v1/auth/logout", r.requireAccount(r.handleLogout))
	r.mux.HandleFunc("/api/v1/extension/bootstrap", r.requireAccount(r.handleExtensionBootstrap))
	r.mux.HandleFunc("/api/v1/overview", r.requireAccount(r.handleOverview))
	r.mux.HandleFunc("/api/v1/accounts", r.requireAccount(r.handleAccounts))
	r.mux.HandleFunc("/api/v1/accounts/", r.requireAccount(r.handleAccountByID))
	r.mux.HandleFunc("/api/v1/groups", r.requireAccount(r.handleGroups))
	r.mux.HandleFunc("/api/v1/groups/", r.requireAccount(r.handleGroupByID))
	r.mux.HandleFunc("/api/v1/node-links", r.requireAccount(r.handleNodeLinks))
	r.mux.HandleFunc("/api/v1/node-access-paths", r.requireAccount(r.handleNodeAccessPaths))
	r.mux.HandleFunc("/api/v1/node-access-paths/", r.requireAccount(r.handleNodeAccessPathByID))
	r.mux.HandleFunc("/api/v1/node-onboarding-tasks", r.requireAccount(r.handleNodeOnboardingTasks))
	r.mux.HandleFunc("/api/v1/node-onboarding-tasks/", r.requireAccount(r.handleNodeOnboardingTaskByID))
	r.mux.HandleFunc("/api/v1/nodes", r.requireAccount(r.handleNodes))
	r.mux.HandleFunc("/api/v1/nodes/", r.requireAccount(r.handleNodeByID))
	r.mux.HandleFunc("/api/v1/node-transports", r.requireAccount(r.handleNodeTransports))
	r.mux.HandleFunc("/api/v1/nodes/connect", r.requireAccount(r.handleNodeConnect))
	r.mux.HandleFunc("/api/v1/nodes/approve/", r.requireAccount(r.handleNodeApprove))
	r.mux.HandleFunc("/api/v1/nodes/bootstrap-token", r.requireAccount(r.handleNodeBootstrapToken))
	r.mux.HandleFunc("/api/v1/nodes/enroll", r.handleNodeEnroll)
	r.mux.HandleFunc("/api/v1/nodes/exchange", r.handleNodeExchange)
	r.mux.HandleFunc("/api/v1/nodes/pending", r.requireAccount(r.handlePendingNodes))
	r.mux.HandleFunc("/api/v1/nodes/scopes", r.requireAccount(r.handleNodeScopes))
	r.mux.HandleFunc("/api/v1/chains", r.requireAccount(r.handleChains))
	r.mux.HandleFunc("/api/v1/chains/validate", r.requireAccount(r.handleChainValidate))
	r.mux.HandleFunc("/api/v1/chains/preview", r.requireAccount(r.handleChainPreview))
	r.mux.HandleFunc("/api/v1/chains/", r.requireAccount(r.handleChainByID))
	r.mux.HandleFunc("/api/v1/route-rules", r.requireAccount(r.handleRouteRules))
	r.mux.HandleFunc("/api/v1/route-rules/match-types", r.requireAccount(r.handleMatchTypes))
	r.mux.HandleFunc("/api/v1/route-rules/validate", r.requireAccount(r.handleRouteRuleValidate))
	r.mux.HandleFunc("/api/v1/route-rules/suggestions", r.requireAccount(r.handleRouteRuleSuggestions))
	r.mux.HandleFunc("/api/v1/route-rules/", r.requireAccount(r.handleRouteRuleByID))
	r.mux.HandleFunc("/api/v1/policies/revisions", r.requireAccount(r.handlePolicyRevisions))
	r.mux.HandleFunc("/api/v1/policies/publish", r.requireAccount(r.handlePolicyPublish))
	r.mux.HandleFunc("/api/v1/certificates", r.requireAccount(r.handleCertificates))
	r.mux.HandleFunc("/api/v1/nodes/health", r.requireAccount(r.handleNodeHealth))
	r.mux.HandleFunc("/api/v1/nodes/health/history", r.requireAccount(r.handleNodeHealthHistory))
	r.mux.HandleFunc("/api/v1/node-agent/policy", r.requireNode(r.handleNodeAgentPolicy))
	r.mux.HandleFunc("/api/v1/node-agent/heartbeat", r.requireNode(r.handleNodeAgentHeartbeat))
	r.mux.HandleFunc("/api/v1/node-agent/cert/renew", r.requireNode(r.handleNodeAgentCertRenew))
	r.mux.HandleFunc("/api/v1/node-agent/transports", r.requireNode(r.handleNodeAgentTransport))
}
