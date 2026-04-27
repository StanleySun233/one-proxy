package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"sync/atomic"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/httpapi"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/network"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/scheduler"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/service"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/setup"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/store"
)

type App struct {
	config config.Config
}

type dynamicHandler struct {
	handler atomic.Value // stores http.Handler
}

func (dh *dynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, _ := dh.handler.Load().(http.Handler)
	if h == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 503, "message": "not_ready"})
		return
	}
	h.ServeHTTP(w, r)
}

func New() *App {
	return &App{
		config: config.Load(),
	}
}

func (a *App) Run() error {
	if config.IsUnconfigured() {
		return a.runSetupMode()
	}
	return a.runFullMode()
}

func (a *App) runSetupMode() error {
	envPath := config.EnvFilePath()
	dh := &dynamicHandler{}

	transitionFn := func() error {
		if err := config.LoadEnvFile(envPath); err != nil {
			return fmt.Errorf("load env: %w", err)
		}
		cfg := config.Load()
		activeStore, err := store.NewMySQLStore(cfg.MySQLDSN)
		if err != nil {
			return fmt.Errorf("mysql store: %w", err)
		}
		if pwd := activeStore.BootstrapAdminPassword(); pwd != "" {
			log.Printf("bootstrap admin account initialized: account=admin")
		}
		controlPlane := service.NewControlPlane(activeStore, cfg)
		sched := scheduler.New(controlPlane, cfg)
		sched.Start()
		fullHandler := httpapi.NewRouter(httpapi.HTTPConfig{
			HTTPAddr:    cfg.HTTPAddr,
			DBBackend:   "mysql",
			EnvFilePath: config.EnvFilePath(),
		}, controlPlane)
		dh.handler.Store(fullHandler)
		log.Printf("control-plane transitioned to full mode")
		return nil
	}

	setupHandler := setup.NewSetupHandler(envPath, transitionFn)
	mux := http.NewServeMux()
	setupHandler.Register(mux)
	wrappedMux := recoveryMiddleware(mux)
	dh.handler.Store(wrappedMux)

	server := &http.Server{
		Addr:    a.config.HTTPAddr,
		Handler: dh,
	}
	log.Printf("control-plane listening on %s (setup mode) localIPs=%v", a.config.HTTPAddr, network.LocalIPs())
	return server.ListenAndServe()
}

func (a *App) runFullMode() error {
	activeStore, err := store.NewMySQLStore(a.config.MySQLDSN)
	if err != nil {
		return err
	}
	if password := activeStore.BootstrapAdminPassword(); password != "" {
		log.Printf("bootstrap admin account initialized: account=admin")
	}
	controlPlane := service.NewControlPlane(activeStore, a.config)
	sched := scheduler.New(controlPlane, a.config)
	sched.Start()
	defer sched.Stop()
	server := &http.Server{
		Addr: a.config.HTTPAddr,
		Handler: httpapi.NewRouter(httpapi.HTTPConfig{
			HTTPAddr:  a.config.HTTPAddr,
			DBBackend: "mysql",
		}, controlPlane),
	}
	log.Printf("control-plane listening on %s localIPs=%v", a.config.HTTPAddr, network.LocalIPs())
	return server.ListenAndServe()
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("http panic setup mode method=%s path=%s err=%v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"code":    500,
					"message": "internal_server_error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
