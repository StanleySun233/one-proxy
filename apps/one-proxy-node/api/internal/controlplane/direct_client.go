package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func (c *Client) ReportDirectCandidates(input domain.ReportDirectCandidatesInput) (domain.ReportDirectCandidatesResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return domain.ReportDirectCandidatesResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/node-agent/direct/candidates", bytes.NewReader(body))
	if err != nil {
		return domain.ReportDirectCandidatesResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.ReportDirectCandidatesResult]
	if err := c.do(req, &envelope); err != nil {
		return domain.ReportDirectCandidatesResult{}, err
	}
	return envelope.Data, nil
}

func (c *Client) FetchDirectLinkPlan() (domain.DirectLinkPlan, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/node-agent/direct/link-plan", nil)
	if err != nil {
		return domain.DirectLinkPlan{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	var envelope responseEnvelope[domain.DirectLinkPlan]
	if err := c.do(req, &envelope); err != nil {
		return domain.DirectLinkPlan{}, err
	}
	return envelope.Data, nil
}

func (c *Client) ReportDirectStatus(input domain.ReportDirectStatusInput) (domain.ReportDirectStatusResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return domain.ReportDirectStatusResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/node-agent/direct/status", bytes.NewReader(body))
	if err != nil {
		return domain.ReportDirectStatusResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	var envelope responseEnvelope[domain.ReportDirectStatusResult]
	if err := c.do(req, &envelope); err != nil {
		return domain.ReportDirectStatusResult{}, err
	}
	return envelope.Data, nil
}
