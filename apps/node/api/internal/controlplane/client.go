package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

type responseEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type ProxyTokenValidation struct {
	Valid           bool    `json:"valid"`
	TenantID        string  `json:"tenantId"`
	ExpiresAt       string  `json:"expiresAt"`
	CacheTTLSeconds int     `json:"cacheTtlSeconds"`
	AllowLocalProxy bool    `json:"allowLocalProxy"`
	ActiveTenantID  *string `json:"activeTenantId"`
}

type ProxyTokenValidationRequest struct {
	TokenHash    string `json:"tokenHash"`
	AccessPathID string `json:"accessPathId"`
	TargetHost   string `json:"targetHost"`
	TargetPort   int    `json:"targetPort"`
	Protocol     string `json:"protocol"`
	RouteID      string `json:"routeId,omitempty"`
}

type NodeAuthValidation struct {
	NodeID string `json:"nodeId"`
}

func New(baseURL string, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/nodes/enroll", bytes.NewReader(body))
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.EnrollNodeResult]
	if err := c.do(req, &envelope); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	return envelope.Data, nil
}

func (c *Client) FetchPolicy() (domain.NodeAgentPolicy, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/node/agent/policy", nil)
	if err != nil {
		return domain.NodeAgentPolicy{}, err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	var envelope responseEnvelope[domain.NodeAgentPolicy]
	if err := c.do(req, &envelope); err != nil {
		return domain.NodeAgentPolicy{}, err
	}
	return envelope.Data, nil
}

func (c *Client) ValidateNodeAuth() (NodeAuthValidation, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/node/agent/auth/validate", nil)
	if err != nil {
		return NodeAuthValidation{}, err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	var envelope responseEnvelope[NodeAuthValidation]
	if err := c.do(req, &envelope); err != nil {
		return NodeAuthValidation{}, err
	}
	return envelope.Data, nil
}

func (c *Client) SendHeartbeat(heartbeatAt time.Time, revision string, listenerStatus map[string]string, certStatus map[string]string) (domain.NodeHealth, error) {
	body, err := json.Marshal(domain.NodeHeartbeatInput{
		HeartbeatTs:      heartbeatAt.UTC().UnixMilli(),
		PolicyRevisionID: revision,
		ListenerStatus:   listenerStatus,
		CertStatus:       certStatus,
	})
	if err != nil {
		return domain.NodeHealth{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/node/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return domain.NodeHealth{}, err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.NodeHealth]
	if err := c.do(req, &envelope); err != nil {
		return domain.NodeHealth{}, err
	}
	return envelope.Data, nil
}

func (c *Client) RenewCertificate(certType string) (domain.NodeCertRenewResult, error) {
	body, err := json.Marshal(domain.NodeCertRenewInput{
		CertType: certType,
	})
	if err != nil {
		return domain.NodeCertRenewResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/node/agent/cert/renew", bytes.NewReader(body))
	if err != nil {
		return domain.NodeCertRenewResult{}, err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.NodeCertRenewResult]
	if err := c.do(req, &envelope); err != nil {
		return domain.NodeCertRenewResult{}, err
	}
	return envelope.Data, nil
}

func (c *Client) UpsertTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return domain.NodeTransport{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/node/agent/transports", bytes.NewReader(body))
	if err != nil {
		return domain.NodeTransport{}, err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.NodeTransport]
	if err := c.do(req, &envelope); err != nil {
		return domain.NodeTransport{}, err
	}
	return envelope.Data, nil
}

func (c *Client) ValidateProxyToken(ctx context.Context, input ProxyTokenValidationRequest) (ProxyTokenValidation, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return ProxyTokenValidation{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/node/agent/proxy/token/validate", bytes.NewReader(body))
	if err != nil {
		return ProxyTokenValidation{}, err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[ProxyTokenValidation]
	if err := c.do(req, &envelope); err != nil {
		return ProxyTokenValidation{}, err
	}
	return envelope.Data, nil
}

func (c *Client) AuthenticateProxyToken(ctx context.Context, tokenHash string) (ProxyTokenValidation, error) {
	body, err := json.Marshal(ProxyTokenValidationRequest{TokenHash: tokenHash})
	if err != nil {
		return ProxyTokenValidation{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/node/agent/proxy/token/authenticate", bytes.NewReader(body))
	if err != nil {
		return ProxyTokenValidation{}, err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[ProxyTokenValidation]
	if err := c.do(req, &envelope); err != nil {
		return ProxyTokenValidation{}, err
	}
	return envelope.Data, nil
}

func (c *Client) ReportProxySessions(ctx context.Context, input domain.ProxySessionMetricsInput) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/node/agent/proxy/sessions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-One-Proxy-Node-Token", c.token)
	req.Header.Set("Content-Type", "application/json")
	var result domain.ProxySessionMetricsResult
	return c.do(req, &result)
}

func (c *Client) ExchangeEnrollment(nodeID string, enrollmentSecret string) (domain.ApproveNodeEnrollmentResult, error) {
	body, err := json.Marshal(domain.ExchangeNodeEnrollmentInput{
		NodeID:           nodeID,
		EnrollmentSecret: enrollmentSecret,
	})
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/nodes/exchange", bytes.NewReader(body))
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.ApproveNodeEnrollmentResult]
	if err := c.do(req, &envelope); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	return envelope.Data, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		var envelope responseEnvelope[map[string]any]
		_ = json.NewDecoder(resp.Body).Decode(&envelope)
		if envelope.Message != "" {
			return errors.New(envelope.Message)
		}
		return errors.New("control_plane_request_failed")
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
