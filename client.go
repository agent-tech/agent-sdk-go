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

const (
	_v2PathPrefix   = "/v2"
	_defaultTimeout = 30 * time.Second
)

// Client calls the v2 payment API.
type Client struct {
	baseURL         string
	httpClient      *http.Client
	authFunc        func(*http.Request)
	hasCustomClient bool   // set by WithHTTPClient
	optErr          error  // first error from an option
}

// NewClient creates a Client for the given baseURL (the API root without /v2).
// At least one auth option (WithBearerAuth or WithAPIKeyAuth) must be provided.
func NewClient(baseURL string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, &ValidationError{Message: "baseURL is required"}
	}
	c := &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: _defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.optErr != nil {
		return nil, c.optErr
	}
	if c.authFunc == nil {
		return nil, &ValidationError{Message: "an auth option is required (use WithBearerAuth or WithAPIKeyAuth)"}
	}
	return c, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	u := c.baseURL + _v2PathPrefix + path

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
	c.authFunc(req)
	return c.httpClient.Do(req)
}

func (c *Client) parseError(resp *http.Response) error {
	var er ErrorResponse
	// Best-effort decode; fall back to resp.Status if body is not JSON.
	_ = json.NewDecoder(resp.Body).Decode(&er)
	msg := er.Message
	if msg == "" {
		msg = er.Error
	}
	if msg == "" {
		msg = resp.Status
	}
	return &APIError{StatusCode: resp.StatusCode, Message: msg}
}

// CreateIntent creates a payment intent (POST /v2/intents).
// Exactly one of req.Email or req.Recipient must be set.
func (c *Client) CreateIntent(ctx context.Context, req *CreateIntentRequest) (*CreateIntentResponse, error) {
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
		return nil, c.parseError(resp)
	}
	var out CreateIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// ExecuteIntent triggers transfer on Base using the Agent wallet (POST /v2/intents/{intent_id}/execute).
// No body or settle_proof required; backend signs and transfers USDC to the intent recipient.
func (c *Client) ExecuteIntent(ctx context.Context, intentID string) (*ExecuteIntentResponse, error) {
	if intentID == "" {
		return nil, &ValidationError{Message: "intent_id is required"}
	}
	resp, err := c.do(ctx, http.MethodPost, "/intents/"+url.PathEscape(intentID)+"/execute", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	var out ExecuteIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// GetIntent returns intent status and receipt (GET /v2/intents?intent_id=...).
func (c *Client) GetIntent(ctx context.Context, intentID string) (*GetIntentResponse, error) {
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
		return nil, c.parseError(resp)
	}
	var out GetIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
