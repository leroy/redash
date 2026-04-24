// Package client is a thin wrapper around the Redash REST API.
//
// It covers the endpoints exposed by the CLI: data sources, schema,
// saved queries (CRUD + execution), ad-hoc queries, jobs, query results,
// dashboards, and users.
//
// The client is intentionally minimal: structs only contain the fields the
// CLI needs, and unknown fields are kept in RawJSON for "get" commands that
// want to print everything.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to a single Redash instance.
type Client struct {
	baseURL *url.URL
	apiKey  string
	http    *http.Client
	ua      string
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the underlying http.Client (useful in tests).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithUserAgent overrides the default User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.ua = ua }
}

// WithInsecure disables TLS certificate verification. Only use this when
// talking to a Redash instance with a self-signed cert.
func WithInsecure() Option {
	return func(c *Client) {
		tr, ok := c.http.Transport.(*http.Transport)
		if !ok || tr == nil {
			tr = http.DefaultTransport.(*http.Transport).Clone()
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
		c.http.Transport = tr
	}
}

// New constructs a Client. baseURL should be the Redash root (e.g.
// "https://redash.example.com"), not including "/api".
func New(baseURL, apiKey string, timeout time.Duration, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base url is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	c := &Client{
		baseURL: u,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: timeout},
		ua:      "redash-cli/dev",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// APIError is returned for non-2xx responses.
type APIError struct {
	Status  int
	Method  string
	Path    string
	Body    string
	Message string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = strings.TrimSpace(e.Body)
	}
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	return fmt.Sprintf("redash api: %s %s -> %d: %s", e.Method, e.Path, e.Status, msg)
}

// Do executes an authenticated request against the Redash API. If body is
// non-nil, it's JSON-encoded. If out is non-nil, the response body is
// JSON-decoded into it. query may be nil.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Key "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.ua)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, u.Path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{Status: resp.StatusCode, Method: method, Path: u.Path, Body: string(raw)}
		// Redash typically returns {"message": "..."}.
		var env struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Message != "" {
			apiErr.Message = env.Message
		}
		return apiErr
	}

	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// BaseURL returns the configured base URL (useful for printing).
func (c *Client) BaseURL() string { return c.baseURL.String() }
