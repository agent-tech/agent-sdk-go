package pay

import (
	"encoding/base64"
	"net/http"
	"time"
)

// Option configures a Client. Pass one or more options to NewClient.
type Option func(*Client)

// WithBearerAuth sets Bearer token authentication using Base64-encoded
// clientID:clientSecret. Both values must be non-empty.
func WithBearerAuth(clientID, clientSecret string) Option {
	return func(c *Client) {
		if clientID == "" || clientSecret == "" {
			if c.optErr == nil {
				c.optErr = &ValidationError{Message: "clientID and clientSecret must not be empty"}
			}
			return
		}
		token := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
		c.authFunc = func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
}

// WithAPIKeyAuth sets header-based authentication using X-Client-ID and
// X-API-Key headers. Both values must be non-empty.
func WithAPIKeyAuth(clientID, apiKey string) Option {
	return func(c *Client) {
		if clientID == "" || apiKey == "" {
			if c.optErr == nil {
				c.optErr = &ValidationError{Message: "clientID and apiKey must not be empty"}
			}
			return
		}
		c.authFunc = func(req *http.Request) {
			req.Header.Set("X-Client-ID", clientID)
			req.Header.Set("X-API-Key", apiKey)
		}
	}
}

// WithHTTPClient replaces the default HTTP client.
// When set, WithTimeout has no effect regardless of option ordering.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
			c.hasCustomClient = true
		}
	}
}

// WithTimeout is a convenience option that sets the timeout on the default
// HTTP client. It is ignored if WithHTTPClient is also provided.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if !c.hasCustomClient {
			c.httpClient.Timeout = d
		}
	}
}
