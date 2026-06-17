package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

const nodeParentProbeTimeout = 2 * time.Second

var errInvalidNodeParentURL = errors.New("invalid_node_parent_url")

func (c *ControlPlane) ProbeNodeParentURL(ctx context.Context, tenantCtx domain.TenantAuthContext, input domain.ProbeNodeParentURLInput) (domain.ProbeNodeParentURLResult, error) {
	if tenantCtx.ActiveTenant.TenantID == "" {
		return domain.ProbeNodeParentURLResult{}, invalidInput("tenant_required")
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return domain.ProbeNodeParentURLResult{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	baseURL, err := normalizeNodeParentURL(input.URL)
	if err != nil {
		return domain.ProbeNodeParentURLResult{}, invalidInput("invalid_parent_url")
	}
	healthURL := nodeParentHealthURL(baseURL)
	result := domain.ProbeNodeParentURLResult{
		URL:       baseURL,
		HealthURL: healthURL,
		Message:   "parent_unreachable",
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return result, nil
	}
	client := &http.Client{
		Timeout: nodeParentProbeTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return result, nil
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return result, nil
	}
	var body struct {
		Status            string `json:"status"`
		Mode              string `json:"mode"`
		ControlPlaneBound *bool  `json:"controlPlaneBound"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body); err != nil {
		return result, nil
	}
	result.Mode = body.Mode
	result.ControlPlaneBound = body.ControlPlaneBound
	if body.Status != "ok" {
		return result, nil
	}
	result.Reachable = true
	result.Message = "parent_reachable"
	return result, nil
}

func normalizeNodeParentURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errInvalidNodeParentURL
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", errInvalidNodeParentURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errInvalidNodeParentURL
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func nodeParentHealthURL(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/healthz"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
