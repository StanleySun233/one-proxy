package httpapi

import (
	"net"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (r *Router) recordBusinessAudit(req *http.Request, input domain.CreateBusinessAuditEventInput) {
	account, ok := accountFromContext(req.Context())
	if ok {
		input.ActorType = "account"
		input.ActorID = account.ID
		input.ActorName = account.Account
	}
	if input.ActorType == "" {
		input.ActorType = "anonymous"
	}
	if tenantCtx, ok := tenantAuthContextFromContext(req.Context()); ok && input.TenantID == "" {
		input.TenantID = tenantCtx.ActiveTenant.TenantID
	}
	if input.ActorIP == "" {
		input.ActorIP = clientIP(req)
	}
	if input.ActorAgent == "" {
		input.ActorAgent = req.UserAgent()
	}
	if input.Outcome == "" {
		input.Outcome = domain.AuditOutcomeSuccess
	}
	_, _ = r.service.RecordBusinessAuditEvent(input)
}

func clientIP(req *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(req.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		return value
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		return host
	}
	return req.RemoteAddr
}
