package app

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/httpapi"
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
	server := &http.Server{
		Addr:    a.config.HTTPAddr,
		Handler: httpapi.NewRouter(a.config),
	}
	return server.ListenAndServe()
}
