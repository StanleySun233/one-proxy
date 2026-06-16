package proxy

import (
	"bytes"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/responsecache"
)

const responseCacheHeader = "X-One-Proxy-Cache"
const responseCacheStoredAtHeader = "X-One-Proxy-Cache-Stored-At"
const responseCacheAgeHeader = "X-One-Proxy-Cache-Age-Seconds"

func (s *Server) storeResponseCache(req *http.Request, body []byte, resp forwardResponse) {
	if s.responseCache == nil || resp.stream != nil {
		return
	}
	if !responsecache.CanStore(req, resp.statusCode, resp.header, len(resp.body)) {
		return
	}
	key := responsecache.Key(req, body)
	if err := s.responseCache.Set(key, responsecache.Entry{
		StatusCode: resp.statusCode,
		Header:     resp.header,
		Body:       resp.body,
	}); err != nil {
		log.Printf("proxy response cache store_failed method=%s target=%s err=%v", req.Method, requestLogTarget(req), err)
		return
	}
	log.Printf("proxy response cache stored method=%s target=%s status=%d bytes=%d", req.Method, requestLogTarget(req), resp.statusCode, len(resp.body))
}

func (s *Server) writeCachedResponseOnError(w http.ResponseWriter, req *http.Request, body []byte, reason string, tracker *proxySessionTracker) bool {
	if s.responseCache == nil {
		return false
	}
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}
	key := responsecache.Key(req, body)
	entry, err := s.responseCache.Get(key)
	if err != nil {
		log.Printf("proxy response cache miss method=%s target=%s reason=%s err=%v", req.Method, requestLogTarget(req), reason, err)
		return false
	}
	resp := cachedForwardResponse(entry)
	tracker.markStatusCode(resp.statusCode)
	tracker.markCache("stale", entry.StoredAt)
	downloadBytes := writeForwardResponse(w, resp)
	tracker.finish(int64(len(body)), downloadBytes, domain.ProxySessionStatusOK, "", "")
	log.Printf("proxy response cache served method=%s target=%s reason=%s status=%d bytes=%d storedAt=%s", req.Method, requestLogTarget(req), reason, entry.StatusCode, len(entry.Body), entry.StoredAt.UTC().Format(time.RFC3339))
	return true
}

func cachedForwardResponse(entry responsecache.Entry) forwardResponse {
	resp := forwardResponse{
		statusCode: entry.StatusCode,
		header:     entry.Header,
		body:       entry.Body,
	}
	resp.header.Set(responseCacheHeader, "stale")
	resp.header.Set(responseCacheStoredAtHeader, entry.StoredAt.UTC().Format(time.RFC3339))
	resp.header.Set(responseCacheAgeHeader, ageSeconds(entry.StoredAt))
	resp.header.Set("Warning", `110 - "Response is stale"`)
	resp.header.Set("Age", ageSeconds(entry.StoredAt))
	if canInjectCacheBanner(resp.header) {
		resp.body = injectCacheBanner(resp.body, entry.StoredAt)
		resp.header.Set("Content-Length", strconv.Itoa(len(resp.body)))
	}
	return resp
}

func canInjectCacheBanner(header http.Header) bool {
	if header.Get("Content-Encoding") != "" {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(header.Get("Content-Type")))
	return strings.HasPrefix(contentType, "text/html") || contentType == ""
}

func injectCacheBanner(body []byte, storedAt time.Time) []byte {
	timestamp := html.EscapeString(storedAt.UTC().Format(time.RFC3339))
	banner := []byte(`<div id="one-proxy-cache-banner" style="position:fixed;left:12px;right:12px;top:12px;z-index:2147483647;padding:10px 12px;border:1px solid #f59e0b;border-radius:8px;background:#fffbeb;color:#78350f;font:13px/1.35 system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;box-shadow:0 12px 28px rgba(15,23,42,.18)">OneProxy cached response - stored at <time datetime="` + timestamp + `">` + timestamp + `</time></div>`)
	lower := bytes.ToLower(body)
	if index := bytes.Index(lower, []byte("<body")); index >= 0 {
		if closeIndex := bytes.IndexByte(body[index:], '>'); closeIndex >= 0 {
			insertAt := index + closeIndex + 1
			result := make([]byte, 0, len(body)+len(banner))
			result = append(result, body[:insertAt]...)
			result = append(result, banner...)
			result = append(result, body[insertAt:]...)
			return result
		}
	}
	result := make([]byte, 0, len(body)+len(banner))
	result = append(result, banner...)
	result = append(result, body...)
	return result
}

func ageSeconds(storedAt time.Time) string {
	age := int(time.Since(storedAt).Seconds())
	if age < 0 {
		age = 0
	}
	return strconvItoa(age)
}

func requestLogTarget(req *http.Request) string {
	if req.URL == nil {
		return req.Host
	}
	if req.URL.IsAbs() {
		target := *req.URL
		target.User = nil
		return target.String()
	}
	if req.Host == "" {
		return req.URL.RequestURI()
	}
	return req.Host + req.URL.RequestURI()
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	i := len(buffer)
	for value > 0 {
		i--
		buffer[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[i:])
}
