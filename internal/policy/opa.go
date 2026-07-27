package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OPAClient struct {
	BaseURL      string
	DecisionPath string
	BearerToken  string
	HTTPClient   *http.Client
}

func (c OPAClient) Evaluate(ctx context.Context, subject, action, resource string) (Decision, error) {
	if c.BaseURL == "" || c.DecisionPath == "" {
		return Decision{}, fmt.Errorf("OPA base URL and decision path are required")
	}
	body, err := json.Marshal(map[string]any{"input": map[string]string{
		"subject":  subject,
		"action":   action,
		"resource": resource,
	}})
	if err != nil {
		return Decision{}, err
	}
	url := strings.TrimRight(c.BaseURL, "/") + "/v1/data/" + strings.Trim(c.DecisionPath, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Decision{}, fmt.Errorf("OPA request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Decision{}, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(data))
	}
	var payload struct {
		Result Decision `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Decision{}, fmt.Errorf("decode OPA response: %w", err)
	}
	if payload.Result.PolicyVersion == "" {
		payload.Result.PolicyVersion = "opa:" + strings.Trim(c.DecisionPath, "/")
	}
	return payload.Result, nil
}
