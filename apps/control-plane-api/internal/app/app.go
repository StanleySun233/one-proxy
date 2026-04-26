package app

import (
	"log"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/httpapi"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/network"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/scheduler"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/service"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/store"
)

type App struct {
	config config.Config
}

func New() *App {
	return &App{
		config: config.Load(),
	}
}

func (a *App) Run() error {
	activeStore, err := store.NewSQLiteStore(a.config.SQLitePath)
	if err != nil {
		log.Printf("sqlite store unavailable, using seed store: %v", err)
		activeStore = nil
	} else if password := activeStore.BootstrapAdminPassword(); password != "" {
		log.Printf("bootstrap admin account initialized: account=admin")
	}
	var dataStore store.Store
	if activeStore == nil {
		seedStore := store.NewSeedStore()
		if password := seedStore.BootstrapAdminPassword(); password != "" {
			log.Printf("seed bootstrap admin account initialized: account=admin")
		}
		dataStore = seedStore
	} else {
		dataStore = activeStore
	}
	controlPlane := service.NewControlPlane(dataStore, a.config)
	sched := scheduler.New(controlPlane, a.config)
	sched.Start()
	defer sched.Stop()
	server := &http.Server{
		Addr:    a.config.HTTPAddr,
		Handler: httpapi.NewRouter(httpapi.HTTPConfig{
			HTTPAddr:   a.config.HTTPAddr,
			SQLitePath: a.config.SQLitePath,
		}, controlPlane),
	}
	log.Printf("control-plane listening on %s localIPs=%v", a.config.HTTPAddr, network.LocalIPs())
	return server.ListenAndServe()
}
