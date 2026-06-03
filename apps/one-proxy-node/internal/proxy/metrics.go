package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
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

func (s *Server) newProxySession(req *http.Request, rule domain.RouteRule, tenantID string) *proxySessionTracker {
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
			ID:         fmt.Sprintf("%s-%d", s.nodeIDGetter(), started.UnixNano()),
			TenantID:   tenantID,
			NodeID:     s.nodeIDGetter(),
			ChainID:    rule.ChainID,
			RouteID:    rule.ID,
			TargetHost: targetHost,
			TargetPort: targetPort,
			Protocol:   protocol,
			StartedAt:  started.Format(time.RFC3339Nano),
		},
	}
}

func (t *proxySessionTracker) finish(uploadBytes int64, downloadBytes int64, status string, errorCode string, errorMessage string) {
	if t == nil {
		return
	}
	t.once.Do(func() {
		ended := time.Now().UTC()
		t.metric.EndedAt = ended.Format(time.RFC3339Nano)
		t.metric.UploadBytes = uploadBytes
		t.metric.DownloadBytes = downloadBytes
		t.metric.LatencyMs = ended.Sub(t.started).Milliseconds()
		t.metric.Status = status
		t.metric.ErrorCode = errorCode
		t.metric.ErrorMessage = errorMessage
		go func(metric domain.ProxySessionMetric) {
			_ = t.reporter.ReportProxySessions(context.Background(), domain.ProxySessionMetricsInput{Sessions: []domain.ProxySessionMetric{metric}})
		}(t.metric)
	})
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
