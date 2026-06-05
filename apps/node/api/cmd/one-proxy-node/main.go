package main

import (
	"bufio"
	"context"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/agentconfig"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/bootstrap"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/cert"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/controlproxy"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/controlrelay"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/direct"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/localconsole"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/network"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/proxy"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/runtime"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/tcpaccess"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/tunnel"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/udpaccess"
	"github.com/quic-go/quic-go"
)

func main() {
	startedAt := time.Now()
	cfg := agentconfig.Load()
	store := policystore.New(cfg.PolicyStatePath)
	interval, err := time.ParseDuration(cfg.HeartbeatInterval)
	if err != nil || interval <= 0 {
		interval = 30 * time.Second
	}
	listenerStatus := map[string]string{"runtime": domain.ListenerStatusUp}
	certStatus := map[string]string{}
	managePublicCert := cfg.PublicCertProvider == "lets_encrypt" && cfg.NodeMode == "edge" && cfg.NodePublicHost != "" && cfg.LetsEncryptEmail != ""
	manager := runtime.New(cfg.RuntimeConfigPath, store, interval, listenerStatus, certStatus, managePublicCert, cfg.NodeJoinPassword, !cfg.NodeJoinPasswordProvided)
	tunnelInterval, tunnelErr := time.ParseDuration(cfg.NodeTunnelHeartbeat)
	if tunnelErr != nil || tunnelInterval <= 0 {
		tunnelInterval = 15 * time.Second
	}
	tunnelRegistry := tunnel.NewRegistry()
	directRegistry := direct.NewRegistry()
	if cfg.ControlPlaneURL != "" && cfg.NodeAccessToken != "" && cfg.NodeID != "" {
		current := manager.Current()
		if err := manager.Attach(runtime.Binding{
			ControlPlaneURL: cfg.ControlPlaneURL,
			NodeID:          cfg.NodeID,
			NodeAccessToken: cfg.NodeAccessToken,
			NodeName:        agentconfig.FirstNonEmpty(cfg.NodeName, current.NodeName),
			NodeMode:        agentconfig.FirstNonEmpty(cfg.NodeMode, current.NodeMode),
			NodeScopeKey:    agentconfig.FirstNonEmpty(cfg.NodeScopeKey, current.NodeScopeKey),
			NodeParentID:    agentconfig.FirstNonEmpty(cfg.NodeParentID, current.NodeParentID),
			NodePublicHost:  agentconfig.FirstNonEmpty(cfg.NodePublicHost, current.NodePublicHost),
			NodePublicPort:  firstPositive(listenPort(cfg.ListenAddr), current.NodePublicPort),
		}); err != nil {
			log.Fatalf("attach runtime binding failed: %v", err)
		}
	} else if cfg.ControlPlaneURL != "" && !manager.Bound() {
		client := controlplane.New(cfg.ControlPlaneURL, cfg.NodeAccessToken)
		if cfg.NodeAccessToken == "" {
			if cfg.EnrollmentSecret == "" {
				if cfg.NodeBootstrapToken == "" {
					log.Fatal("missing NODE_ACCESS_TOKEN or NODE_ENROLLMENT_SECRET or NODE_BOOTSTRAP_TOKEN")
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
	proxyAuth := proxy.AuthConfig{CacheTTL: parseDurationOrDefault(cfg.NodeProxyTokenCacheTTL, 24*time.Hour)}
	if manager.Bound() {
		current := manager.Current()
		proxyAuth.Validator = proxyTokenValidator(current.ControlPlaneURL, current.NodeAccessToken)
	}
	proxyAuthorizer := proxy.NewTokenAuthorizer(proxyAuth)
	proxyHandler, err := proxy.NewServerWithAuthorizer(store, manager.NodeID, tunnelRegistry, cfg.NodeReverseTargetURL, proxy.AuthConfig{
		Validator: proxyAuth.Validator,
		CacheTTL:  proxyAuth.CacheTTL,
	}, proxyAuthorizer)
	if err != nil {
		log.Fatalf("init proxy server failed: %v", err)
	}
	if manager.Bound() {
		current := manager.Current()
		proxyHandler.SetProxySessionReporter(controlplane.New(current.ControlPlaneURL, current.NodeAccessToken))
	}
	proxyHandler.SetDirectStreamOpener(directRegistry)
	mux := http.NewServeMux()
	localConsole := localconsole.New(manager, store, cfg, startedAt)
	consoleWeb := localconsole.WebHandler(cfg.NodeConsoleWebRoot, localConsole.Authenticated)
	mux.Handle("/api/local/", localConsole)
	mux.Handle("/", nodeRouteSplitHandler(consoleWeb, proxyHandler))
	httpHandler := proxyAwareHandler(proxyHandler, mux)
	mux.Handle("/api/control/relay/probe", controlrelay.NewProbeHandler(tunnelRegistry, func() string {
		if !manager.Bound() {
			return ""
		}
		return manager.Current().NodeID
	}))
	mux.Handle("/api/node/bootstrap/attach", bootstrap.New(cfg.ListenAddr, cfg.HTTPSListenAddr, manager))
	mux.Handle(cfg.NodeTunnelPath, tunnel.NewServer(manager, tunnelRegistry))
	if manager.Bound() {
		current := manager.Current()
		forwarder, err := controlproxy.New(current.ControlPlaneURL)
		if err != nil {
			log.Fatalf("init control proxy failed: %v", err)
		}
		mux.Handle("/api/nodes/enroll", forwarder)
		mux.Handle("/api/nodes/exchange", forwarder)
		mux.Handle("/api/node/agent/policy", forwarder)
		mux.Handle("/api/node/agent/auth/validate", forwarder)
		mux.Handle("/api/node/agent/heartbeat", forwarder)
		mux.Handle("/api/node/agent/cert/renew", forwarder)
		mux.Handle("/api/node/agent/transports", forwarder)
		mux.Handle("/api/node/agent/proxy/sessions", forwarder)
		mux.Handle("/api/node/agent/direct/candidates", forwarder)
		mux.Handle("/api/node/agent/direct/link/plan", forwarder)
		mux.Handle("/api/node/agent/direct/status", forwarder)
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
		certStatus["public"] = domain.CertStatusHealthy
		go func() {
			httpsServer := &http.Server{
				Addr:      cfg.HTTPSListenAddr,
				Handler:   mux,
				TLSConfig: certManager.TLSConfig(),
			}
			log.Fatal(httpsServer.ListenAndServeTLS("", ""))
		}()
	}
	tunnelController := tunnel.NewController(manager, tunnelRegistry, cfg.NodeTunnelPath, tunnelInterval)
	go tunnelController.Run()
	startDirectManager(cfg, manager, directRegistry)
	go manager.Run()
	if cfg.TCPAccessListenAddr != "" {
		go tcpaccess.ListenAndServe(cfg.TCPAccessListenAddr, tcpaccess.New(proxyAuthorizer, tunnelRegistry))
	}
	if cfg.UDPAccessListenAddr != "" {
		go udpaccess.ListenAndServe(cfg.UDPAccessListenAddr, udpaccess.New(proxyAuthorizer))
	}
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: withObservability(httpHandler),
	}
	log.Printf("proxy-node listening on http=%s https=%s localIPs=%v", cfg.ListenAddr, cfg.HTTPSListenAddr, network.LocalIPs())
	log.Fatal(server.ListenAndServe())
}

func startDirectManager(cfg agentconfig.Config, manager *runtime.Manager, registry *direct.Registry) {
	if cfg.NodeDirectListenAddr == "" || !manager.Bound() {
		return
	}
	addr, err := net.ResolveUDPAddr("udp", cfg.NodeDirectListenAddr)
	if err != nil {
		log.Printf("direct transport disabled: resolve listen addr failed: %v", err)
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("direct transport disabled: listen failed: %v", err)
		return
	}
	quicTransport := &quic.Transport{Conn: conn}
	if err := registry.AttachQUICTransport(quicTransport); err != nil {
		log.Printf("direct transport disabled: quic listen failed: %v", err)
		return
	}
	interval := parseDurationOrDefault(cfg.NodeDirectRefreshInterval, 30*time.Second)
	current := manager.Current()
	client := controlplane.New(current.ControlPlaneURL, current.NodeAccessToken)
	registry.SetClientSessionValidator(clientDirectSessionValidator{client: client})
	directManager := direct.NewManager(direct.QUICPacketIO{Transport: quicTransport}, direct.CandidateGatherer{
		STUNServers: splitCSV(cfg.NodeDirectSTUNServers),
		Timeout:     3 * time.Second,
	}, client, registry)
	log.Printf("direct transport listening on udp=%s stun=%s", conn.LocalAddr().String(), cfg.NodeDirectSTUNServers)
	go registry.RunQUICServer(context.Background())
	go directManager.Run(context.Background(), interval, func(err error) {
		log.Printf("direct transport refresh failed: %v", err)
	})
}

type clientDirectSessionValidator struct {
	client *controlplane.Client
}

func (v clientDirectSessionValidator) ValidateClientDirectSession(ctx context.Context, request direct.ClientSessionValidationRequest) (direct.ClientSessionValidationResult, error) {
	result, err := v.client.ValidateClientDirectSession(domain.ClientDirectSessionValidateInput{
		SessionID:  request.SessionID,
		PunchToken: request.PunchToken,
		TargetHost: request.TargetHost,
		TargetPort: request.TargetPort,
	})
	if err != nil {
		return direct.ClientSessionValidationResult{}, err
	}
	return direct.ClientSessionValidationResult{
		Valid:      result.Valid,
		TargetHost: result.TargetHost,
		TargetPort: result.TargetPort,
	}, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

type panelProxyTokenValidator struct {
	client *controlplane.Client
}

func proxyTokenValidator(controlPlaneURL string, nodeAccessToken string) proxy.TokenValidator {
	if controlPlaneURL == "" || nodeAccessToken == "" {
		return nil
	}
	return panelProxyTokenValidator{client: controlplane.New(controlPlaneURL, nodeAccessToken)}
}

func (v panelProxyTokenValidator) ValidateProxyToken(ctx context.Context, tokenHash string) (proxy.TokenValidation, error) {
	result, err := v.client.ValidateProxyToken(ctx, tokenHash)
	if err != nil {
		return proxy.TokenValidation{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
	if err != nil {
		return proxy.TokenValidation{Valid: result.Valid, AllowLocalProxy: result.AllowLocalProxy, TenantID: stringValue(result.ActiveTenantID)}, nil
	}
	return proxy.TokenValidation{
		Valid:           result.Valid,
		ExpiresAt:       expiresAt,
		CacheTTL:        time.Duration(result.CacheTTLSeconds) * time.Second,
		AllowLocalProxy: result.AllowLocalProxy,
		TenantID:        stringValue(result.ActiveTenantID),
	}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseDurationOrDefault(value string, fallback time.Duration) time.Duration {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return fallback
	}
	return duration
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
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

func proxyAwareHandler(proxyHandler http.Handler, mux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodConnect {
			proxyHandler.ServeHTTP(w, req)
			return
		}
		mux.ServeHTTP(w, req)
	})
}

func nodeRouteSplitHandler(consoleWeb http.Handler, proxyHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL != nil && req.URL.IsAbs() {
			proxyHandler.ServeHTTP(w, req)
			return
		}
		if localconsole.ConsoleRoute(req.URL.Path) || localconsole.StaticAssetRoute(req.URL.Path) {
			consoleWeb.ServeHTTP(w, req)
			return
		}
		proxyHandler.ServeHTTP(w, req)
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

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
