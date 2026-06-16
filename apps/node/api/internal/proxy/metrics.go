package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

type ProxySessionReporter interface {
	ReportProxySessions(context.Context, domain.ProxySessionMetricsInput) error
}

type proxySessionTracker struct {
	reporter ProxySessionReporter
	metric   domain.ProxySessionMetric
	started  time.Time
	once     sync.Once
}

type countingReadCloser struct {
	io.ReadCloser
	bytes *int64
}

type countingWriter struct {
	io.Writer
	bytes *int64
}

func (s *Server) SetProxySessionReporter(reporter ProxySessionReporter) {
	s.metricsReporter = reporter
}

func (s *Server) newProxySession(req *http.Request, rule domain.RouteRule, tenantID string, policyRevision string) *proxySessionTracker {
	if s.metricsReporter == nil {
		return nil
	}
	targetHost, targetPort := targetAddress(req)
	protocol := domain.ProxySessionProtocolHTTP
	if req.Method == http.MethodConnect {
		protocol = domain.ProxySessionProtocolConnect
	}
	started := time.Now().UTC()
	return &proxySessionTracker{
		reporter: s.metricsReporter,
		started:  started,
		metric: domain.ProxySessionMetric{
			ID:                 fmt.Sprintf("%s-%d", s.nodeIDGetter(), started.UnixNano()),
			TenantID:           tenantID,
			NodeID:             s.nodeIDGetter(),
			ChainID:            rule.ChainID,
			ScopeID:            rule.DestinationScope,
			RouteID:            rule.ID,
			GovernanceMode:     "enforce",
			PolicyRevision:     policyRevision,
			MatchedRuleID:      rule.ID,
			MatchedRuleType:    rule.MatchType,
			MatchedRulePattern: rule.MatchValue,
			MatchedAction:      rule.ActionType,
			DecisionSource:     routeDecisionSource(rule),
			TargetHost:         targetHost,
			TargetPort:         targetPort,
			Protocol:           protocol,
			StartedAt:          started.Format(time.RFC3339Nano),
			ReceiveTSMs:        started.UnixMilli(),
		},
	}
}

func routeDecisionSource(rule domain.RouteRule) string {
	if rule.ID != "" {
		return "policy"
	}
	return "default"
}

func (s *Server) newReverseProxySession(req *http.Request, tenantID string) *proxySessionTracker {
	if s.metricsReporter == nil {
		return nil
	}
	targetHost, targetPort := targetURLAddress(s.reverseTarget)
	started := time.Now().UTC()
	return &proxySessionTracker{
		reporter: s.metricsReporter,
		started:  started,
		metric: domain.ProxySessionMetric{
			ID:          fmt.Sprintf("%s-%d", s.nodeIDGetter(), started.UnixNano()),
			TenantID:    tenantID,
			NodeID:      s.nodeIDGetter(),
			TargetHost:  targetHost,
			TargetPort:  targetPort,
			Protocol:    domain.ProxySessionProtocolHTTP,
			StartedAt:   started.Format(time.RFC3339Nano),
			ReceiveTSMs: started.UnixMilli(),
		},
	}
}

func targetURLAddress(target *url.URL) (string, int) {
	if target == nil {
		return "", 0
	}
	port := target.Port()
	if port != "" {
		value, _ := strconv.Atoi(port)
		return target.Hostname(), value
	}
	switch target.Scheme {
	case "https", "wss":
		return target.Hostname(), 443
	default:
		return target.Hostname(), 80
	}
}

func (t *proxySessionTracker) markForward() {
	if t == nil || t.metric.ForwardTSMs > 0 {
		return
	}
	now := time.Now().UTC()
	t.metric.ForwardTSMs = now.UnixMilli()
	t.metric.NodeProcessMs = now.Sub(t.started).Milliseconds()
}

func (t *proxySessionTracker) markResponseReceive() {
	if t == nil || t.metric.ResponseReceiveTSMs > 0 {
		return
	}
	t.metric.ResponseReceiveTSMs = time.Now().UTC().UnixMilli()
}

func (t *proxySessionTracker) markStatusCode(statusCode int) {
	if t == nil || statusCode <= 0 {
		return
	}
	t.metric.StatusCode = statusCode
}

func (t *proxySessionTracker) markCache(status string, storedAt time.Time) {
	if t == nil || status == "" {
		return
	}
	t.metric.CacheStatus = status
	if !storedAt.IsZero() {
		t.metric.CacheStoredAt = storedAt.UTC().Format(time.RFC3339)
	}
}

func (t *proxySessionTracker) addLinkTiming(fromNodeID string, toNodeID string, started time.Time) {
	if t == nil || fromNodeID == "" || toNodeID == "" {
		return
	}
	ended := time.Now().UTC()
	t.metric.LinkTimings = append(t.metric.LinkTimings, domain.ProxyLinkTiming{
		FromNodeID: fromNodeID,
		ToNodeID:   toNodeID,
		RTTMs:      ended.Sub(started).Milliseconds(),
		SampleTSMs: ended.UnixMilli(),
		Count:      1,
	})
}

func (t *proxySessionTracker) finish(uploadBytes int64, downloadBytes int64, status string, errorCode string, errorMessage string) {
	if t == nil {
		return
	}
	t.once.Do(func() {
		ended := time.Now().UTC()
		if t.metric.ForwardTSMs == 0 {
			t.markForward()
		}
		if t.metric.ResponseReceiveTSMs == 0 {
			t.metric.ResponseReceiveTSMs = ended.UnixMilli()
		}
		t.metric.EndedAt = ended.Format(time.RFC3339Nano)
		t.metric.UploadBytes = uploadBytes
		t.metric.DownloadBytes = downloadBytes
		t.metric.LatencyMs = ended.Sub(t.started).Milliseconds()
		t.metric.ResponseForwardTSMs = ended.UnixMilli()
		t.metric.ResponseProcessMs = t.metric.ResponseForwardTSMs - t.metric.ResponseReceiveTSMs
		t.metric.NodeTimings = []domain.ProxyNodeTiming{{
			NodeID:               t.metric.NodeID,
			ProcessAvgMs:         t.metric.NodeProcessMs,
			ResponseProcessAvgMs: t.metric.ResponseProcessMs,
			SampleTSMs:           t.metric.ResponseForwardTSMs,
			Count:                1,
		}}
		if t.metric.StatusCode <= 0 {
			t.metric.StatusCode = statusCodeForProxyError(errorCode)
		}
		t.metric.Status = status
		t.metric.ErrorCode = errorCode
		t.metric.ErrorMessage = errorMessage
		go func(metric domain.ProxySessionMetric) {
			_ = t.reporter.ReportProxySessions(context.Background(), domain.ProxySessionMetricsInput{Sessions: []domain.ProxySessionMetric{metric}})
		}(t.metric)
	})
}

func statusCodeForProxyError(errorCode string) int {
	switch errorCode {
	case "":
		return http.StatusOK
	case proxyErrorRouteNotFound:
		return http.StatusForbidden
	case proxyErrorProxyAuthRequired:
		return http.StatusProxyAuthRequired
	case proxyErrorReverseAuthRequired:
		return http.StatusUnauthorized
	case proxyErrorUnsupportedRouteAction:
		return http.StatusBadRequest
	case proxyErrorHijackNotSupported:
		return http.StatusInternalServerError
	default:
		return http.StatusBadGateway
	}
}

func (r countingReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	atomic.AddInt64(r.bytes, int64(n))
	return n, err
}

func (w countingWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	atomic.AddInt64(w.bytes, int64(n))
	return n, err
}

func bridgeTunnelWithMetrics(clientConn net.Conn, backendConn net.Conn, backendReader io.Reader, tracker *proxySessionTracker) {
	var uploadBytes int64
	var downloadBytes int64
	var wait sync.WaitGroup
	var closeOnce sync.Once
	closeBoth := func() {
		closeOnce.Do(func() {
			_ = clientConn.Close()
			_ = backendConn.Close()
		})
	}
	wait.Add(2)
	go func() {
		defer wait.Done()
		_, _ = io.Copy(countingWriter{Writer: backendConn, bytes: &uploadBytes}, clientConn)
		closeBoth()
	}()
	go func() {
		defer wait.Done()
		_, _ = io.Copy(countingWriter{Writer: clientConn, bytes: &downloadBytes}, backendReader)
		closeBoth()
	}()
	go func() {
		wait.Wait()
		tracker.finish(atomic.LoadInt64(&uploadBytes), atomic.LoadInt64(&downloadBytes), domain.ProxySessionStatusOK, "", "")
	}()
}
