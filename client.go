package caribbeancinemas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Default configuration for the Caribbean Cinemas circuit. These are the values
// the public website sends; they rarely need changing.
const (
	DefaultEndpoint   = "https://home.caribbeancinemas.com/graphql"
	DefaultCircuitID  = "2"
	DefaultClientType = "consumer"
	// DefaultSiteID is the circuit "home page" virtual site. It scopes
	// header-sensitive queries (e.g. Screens, TicketTypes) when no specific
	// theater is given.
	DefaultSiteID    = "96"
	DefaultUserAgent = "caribbeancinemas-go/0.1 (+https://github.com/jasielrt/caribbeancinemas-go)"
)

// Client is a read-only client for the Caribbean Cinemas GraphQL API. It is safe
// for concurrent use by multiple goroutines. Create one with New.
type Client struct {
	endpoint   string
	circuitID  string
	clientType string
	siteID     string
	userAgent  string
	httpClient *http.Client
}

// Option configures a Client. Pass options to New.
type Option func(*Client)

// WithHTTPClient sets a custom http.Client (for timeouts, transports, tracing,
// etc.). By default a client with a 30s timeout is used.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithEndpoint overrides the GraphQL endpoint URL.
func WithEndpoint(url string) Option {
	return func(c *Client) { c.endpoint = url }
}

// WithSiteID sets the default site-id header used to scope header-sensitive
// queries. Most methods that care about a specific theater take a site ID
// argument directly; this only sets the default.
func WithSiteID(siteID string) Option {
	return func(c *Client) { c.siteID = siteID }
}

// WithCircuitID overrides the circuit-id header (default "2").
func WithCircuitID(circuitID string) Option {
	return func(c *Client) { c.circuitID = circuitID }
}

// WithUserAgent sets the User-Agent header. Please keep it identifiable.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// New creates a Client with sensible defaults for the Caribbean Cinemas circuit.
func New(opts ...Option) *Client {
	c := &Client{
		endpoint:   DefaultEndpoint,
		circuitID:  DefaultCircuitID,
		clientType: DefaultClientType,
		siteID:     DefaultSiteID,
		userAgent:  DefaultUserAgent,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func (c *Client) execute(ctx context.Context, siteID, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: vars})
	if err != nil {
		return fmt.Errorf("caribbeancinemas: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("caribbeancinemas: build request: %w", err)
	}
	if siteID == "" {
		siteID = c.siteID
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("circuit-id", c.circuitID)
	req.Header.Set("client-type", c.clientType)
	req.Header.Set("site-id", siteID)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("caribbeancinemas: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("caribbeancinemas: read response: %w", err)
	}
	if resp.StatusCode >= 500 {
		return &APIError{Message: fmt.Sprintf("server returned HTTP %d", resp.StatusCode)}
	}

	var envelope graphQLResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("caribbeancinemas: decode response: %w", err)
	}
	if envelope.Error != nil {
		return &APIError{Code: envelope.Error.Code, Message: envelope.Error.Message}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("caribbeancinemas: decode data: %w", err)
	}
	return nil
}
