package tunnel

import (
	"net"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/runtime"
)

type Controller struct {
	manager           *runtime.Manager
	registry          *Registry
	tunnelPath        string
	heartbeatInterval time.Duration
	writeMu           sync.Mutex
	streamsMu         sync.RWMutex
	streams           map[string]net.Conn
}

func NewController(manager *runtime.Manager, registry *Registry, tunnelPath string, heartbeatInterval time.Duration) *Controller {
	return &Controller{
		manager:           manager,
		registry:          registry,
		tunnelPath:        tunnelPath,
		heartbeatInterval: heartbeatInterval,
		streams:           make(map[string]net.Conn),
	}
}
