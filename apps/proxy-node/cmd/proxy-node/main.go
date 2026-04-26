package main

import (
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/agentconfig"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/bootstrap"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/cert"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/controlproxy"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/controlrelay"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/network"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/policystore"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/proxy"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/runtime"
)

func main() {
	cfg := agentconfig.Load()
	store := policystore.New(cfg.PolicyStatePath)
	interval, err := time.ParseDuration(cfg.HeartbeatInterval)
	if err != nil || interval <= 0 {
		interval = 30 * time.Second
	}
	listenerStatus := map[string]string{"http": "healthy"}
	certStatus := map[string]string{"internal": "healthy"}
	managePublicCert := cfg.PublicCertProvider == "lets_encrypt" && cfg.NodeMode == "edge" && cfg.NodePublicHost != "" && cfg.LetsEncryptEmail != ""
	manager := runtime.New(cfg.RuntimeConfigPath, store, interval, listenerStatus, certStatus, managePublicCert)
	if cfg.ControlPlaneURL != "" && cfg.NodeAccessToken != "" && cfg.NodeID != "" {
		if err := manager.Attach(runtime.Binding{
			ControlPlaneURL: cfg.ControlPlaneURL,
			NodeID:          cfg.NodeID,
			NodeAccessToken: cfg.NodeAccessToken,
			NodeName:        cfg.NodeName,
			NodeMode:        cfg.NodeMode,
			NodeScopeKey:    cfg.NodeScopeKey,
			NodeParentID:    cfg.NodeParentID,
			NodePublicHost:  cfg.NodePublicHost,
			NodePublicPort:  listenPort(cfg.ListenAddr),
		}); err != nil {
			log.Fatalf("attach runtime binding failed: %v", err)
		}
	} else if cfg.ControlPlaneURL != "" {
		client := controlplane.New(cfg.ControlPlaneURL, cfg.NodeAccessToken)
		if cfg.NodeAccessToken == "" {
			if cfg.EnrollmentSecret == "" {
				if cfg.NodeBootstrapToken == "" {
					log.Fatal("missing NODE_ACCESS_TOKEN or NODE_ENROLLMENT_SECRET or NODE_BOOTSTRAP_TOKEN")
				}
				if cfg.NodeName == "" || cfg.NodeScopeKey == "" {
					log.Fatal("missing NODE_NAME or NODE_SCOPE_KEY for bootstrap enrollment")
				}
				enroll, err := client.EnrollNode(domain.EnrollNodeInput{
					Token:        cfg.NodeBootstrapToken,
					Name:         cfg.NodeName,
					Mode:         cfg.NodeMode,
					ScopeKey:     cfg.NodeScopeKey,
					ParentNodeID: cfg.NodeParentID,
					PublicHost:   cfg.NodePublicHost,
					PublicPort:   listenPort(cfg.ListenAddr),
				})
				if err != nil {
					log.Fatalf("enroll node failed: %v", err)
				}
				cfg.NodeID = enroll.Node.ID
				cfg.EnrollmentSecret = enroll.EnrollmentSecret
				log.Printf("node enrolled nodeID=%s approvalState=%s", cfg.NodeID, enroll.ApprovalState)
			}
			if cfg.NodeID == "" {
				log.Fatal("missing NODE_ID after enrollment bootstrap")
			}
			exchange, err := waitForApproval(client, cfg.NodeID, cfg.EnrollmentSecret)
			if err != nil {
				log.Fatalf("exchange enrollment failed: %v", err)
			}
			if err := manager.Attach(runtime.Binding{
				ControlPlaneURL: cfg.ControlPlaneURL,
				NodeID:          exchange.Node.ID,
				NodeAccessToken: exchange.AccessToken,
				NodeName:        exchange.Node.Name,
				NodeMode:        exchange.Node.Mode,
				NodeScopeKey:    exchange.Node.ScopeKey,
				NodeParentID:    exchange.Node.ParentNodeID,
				NodePublicHost:  exchange.Node.PublicHost,
				NodePublicPort:  exchange.Node.PublicPort,
			}); err != nil {
				log.Fatalf("attach runtime binding failed: %v", err)
			}
		}
	}
	proxyHandler := proxy.NewServer(store, manager.NodeID)
	mux := http.NewServeMux()
	mux.Handle("/", proxyHandler)
	httpHandler := http.Handler(mux)
	mux.Handle("/api/v1/control-relay/probe", controlrelay.NewProbeHandler())
	mux.Handle("/api/v1/node/bootstrap/attach", bootstrap.New(cfg.NodeJoinPassword, cfg.ListenAddr, cfg.HTTPSListenAddr, manager))
	if manager.Bound() {
		current := manager.Current()
		forwarder, err := controlproxy.New(current.ControlPlaneURL)
		if err != nil {
			log.Fatalf("init control proxy failed: %v", err)
		}
		mux.Handle("/api/v1/nodes/enroll", forwarder)
		mux.Handle("/api/v1/nodes/exchange", forwarder)
		mux.Handle("/api/v1/node-agent/policy", forwarder)
		mux.Handle("/api/v1/node-agent/heartbeat", forwarder)
		mux.Handle("/api/v1/node-agent/cert/renew", forwarder)
		log.Printf("proxy-node bound nodeID=%s controlPlaneURL=%s", current.NodeID, current.ControlPlaneURL)
	} else {
		log.Printf("proxy-node starting without control plane binding localIPs=%v", network.LocalIPs())
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if manager.Bound() {
			_, _ = w.Write([]byte(`{"status":"ok","mode":"proxy-node","controlPlaneBound":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","mode":"proxy-node","controlPlaneBound":false}`))
	})
	if managePublicCert {
		certManager, err := cert.NewLetsEncryptManager(cfg.LetsEncryptEmail, cfg.LetsEncryptCacheDir, cfg.NodePublicHost)
		if err != nil {
			log.Fatalf("init letsencrypt manager failed: %v", err)
		}
		httpHandler = certManager.HTTPHandler(mux)
		listenerStatus["https"] = "healthy"
		certStatus["public"] = "managed"
		go func() {
			httpsServer := &http.Server{
				Addr:      cfg.HTTPSListenAddr,
				Handler:   mux,
				TLSConfig: certManager.TLSConfig(),
			}
			log.Fatal(httpsServer.ListenAndServeTLS("", ""))
		}()
	}
	go manager.Run()
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: withObservability(httpHandler),
	}
	log.Printf("proxy-node listening on http=%s https=%s localIPs=%v", cfg.ListenAddr, cfg.HTTPSListenAddr, network.LocalIPs())
	log.Fatal(server.ListenAndServe())
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
				log.Printf("proxy-node panic method=%s path=%s err=%v\n%s", req.Method, req.URL.Path, recovered, debug.Stack())
				http.Error(sw, "internal_server_error", http.StatusInternalServerError)
				return
			}
			if sw.status != http.StatusOK {
				log.Printf("proxy-node request method=%s path=%s status=%d duration=%s", req.Method, req.URL.Path, sw.status, time.Since(startedAt))
			}
		}()
		next.ServeHTTP(sw, req)
	})
}

func waitForApproval(client *controlplane.Client, nodeID string, enrollmentSecret string) (domain.ApproveNodeEnrollmentResult, error) {
	for {
		result, err := client.ExchangeEnrollment(nodeID, enrollmentSecret)
		if err == nil {
			return result, nil
		}
		if !strings.Contains(err.Error(), "node_enrollment_pending") {
			return domain.ApproveNodeEnrollmentResult{}, err
		}
		log.Printf("node enrollment pending nodeID=%s", nodeID)
		time.Sleep(5 * time.Second)
	}
}

func listenPort(addr string) int {
	parts := strings.Split(addr, ":")
	if len(parts) == 0 {
		return 0
	}
	value := parts[len(parts)-1]
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return port
}
