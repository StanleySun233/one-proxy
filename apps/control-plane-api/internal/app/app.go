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
