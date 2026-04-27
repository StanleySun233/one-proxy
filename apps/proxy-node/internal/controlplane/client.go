package controlplane

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/domain"
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
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/nodes/enroll", bytes.NewReader(body))
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
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/v1/node-agent/policy", nil)
	if err != nil {
		return domain.NodeAgentPolicy{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	var envelope responseEnvelope[domain.NodeAgentPolicy]
	if err := c.do(req, &envelope); err != nil {
		return domain.NodeAgentPolicy{}, err
	}
	return envelope.Data, nil
}

func (c *Client) SendHeartbeat(revision string, listenerStatus map[string]string, certStatus map[string]string) (domain.NodeHealth, error) {
	body, err := json.Marshal(domain.NodeHeartbeatInput{
		PolicyRevisionID: revision,
		ListenerStatus:   listenerStatus,
		CertStatus:       certStatus,
	})
	if err != nil {
		return domain.NodeHealth{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/node-agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return domain.NodeHealth{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
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
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/node-agent/cert/renew", bytes.NewReader(body))
	if err != nil {
		return domain.NodeCertRenewResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
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
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/node-agent/transports", bytes.NewReader(body))
	if err != nil {
		return domain.NodeTransport{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.NodeTransport]
	if err := c.do(req, &envelope); err != nil {
		return domain.NodeTransport{}, err
	}
	return envelope.Data, nil
}

func (c *Client) ExchangeEnrollment(nodeID string, enrollmentSecret string) (domain.ApproveNodeEnrollmentResult, error) {
	body, err := json.Marshal(domain.ExchangeNodeEnrollmentInput{
		NodeID:           nodeID,
		EnrollmentSecret: enrollmentSecret,
	})
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/nodes/exchange", bytes.NewReader(body))
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
