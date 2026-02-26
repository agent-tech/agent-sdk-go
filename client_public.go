package pay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PublicClient calls the public payment API (no auth, /api prefix).
// Use when the integrator has the payer's wallet and can sign X402 / submit settle_proof.
type PublicClient struct {
	baseURL         string
	httpClient      *http.Client
	hasCustomClient bool
}

// PublicOption configures a PublicClient.
type PublicOption func(*PublicClient)

// NewPublicClient creates a PublicClient for the given baseURL (API root without /api).
// No authentication is required.
func NewPublicClient(baseURL string, opts ...PublicOption) (*PublicClient, error) {
	if baseURL == "" {
		return nil, &ValidationError{Message: "baseURL is required"}
	}
	c := &PublicClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: _defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// WithPublicHTTPClient replaces the default HTTP client.
// When set, WithPublicTimeout has no effect regardless of option ordering.
func WithPublicHTTPClient(hc *http.Client) PublicOption {
	return func(c *PublicClient) {
		if hc != nil {
			c.httpClient = hc
			c.hasCustomClient = true
		}
	}
}

// WithPublicTimeout sets the timeout on the default HTTP client.
// Ignored if WithPublicHTTPClient is also provided.
func WithPublicTimeout(d time.Duration) PublicOption {
	return func(c *PublicClient) {
		if !c.hasCustomClient {
			c.httpClient.Timeout = d
		}
	}
}

func (c *PublicClient) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	u := c.baseURL + _apiPathPrefix + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

// CreateIntent creates a payment intent (POST /api/intents).
// Exactly one of req.Email or req.Recipient must be set.
func (c *PublicClient) CreateIntent(ctx context.Context, req *CreateIntentRequest) (*CreateIntentResponse, error) {
	if req == nil {
		return nil, &ValidationError{Message: "CreateIntentRequest is nil"}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := c.do(ctx, http.MethodPost, "/intents", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}
	var out CreateIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// SubmitProof submits settle_proof after the payer has completed X402 payment on the source chain (POST /api/intents/{intent_id}).
func (c *PublicClient) SubmitProof(ctx context.Context, intentID, settleProof string) (*SubmitProofResponse, error) {
	if intentID == "" {
		return nil, &ValidationError{Message: "intent_id is required"}
	}
	if settleProof == "" {
		return nil, &ValidationError{Message: "settle_proof is required"}
	}
	body, err := json.Marshal(map[string]string{"settle_proof": settleProof})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	path := "/intents/" + url.PathEscape(intentID)
	resp, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}
	var out SubmitProofResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// GetIntent returns intent status and receipt (GET /api/intents?intent_id=...).
func (c *PublicClient) GetIntent(ctx context.Context, intentID string) (*GetIntentResponse, error) {
	if intentID == "" {
		return nil, &ValidationError{Message: "intent_id is required"}
	}
	path := "/intents?intent_id=" + url.QueryEscape(intentID)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}
	var out GetIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
