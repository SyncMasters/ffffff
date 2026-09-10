// Package hibp integrates the official authenticated email breach API.
// Password lookup is intentionally a separate, unimplemented capability.
package hibp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/johan-larp/agentsearch/internal/security"
)

const DefaultBaseURL = "https://haveibeenpwned.com/api/v3"
const userAgent = "AgentSearch (https://github.com/johan-larp/AgentSearch)"

// Client is safe for concurrent use. It does not log requests or retry failures.
// Inject a standard, non-retrying client from network.NewServiceClient.
type Client struct {
	http    *http.Client
	baseURL string
	key     security.Secret
}

func NewClient(baseURL string, key security.Secret, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(key.Reveal()) == "" {
		return nil, &Error{Kind: MissingKey}
	}
	if strings.ContainsAny(key.Reveal(), "\r\n") {
		return nil, &Error{Kind: Unauthorized}
	}
	if httpClient == nil || httpClient.Timeout <= 0 {
		return nil, fmt.Errorf("HIBP requires an HTTP client with a positive timeout")
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, fmt.Errorf("invalid HIBP base URL; credentials, queries and fragments are not allowed")
	}
	loopback := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback != nil && loopback.IsLoopback()) {
		return nil, fmt.Errorf("HIBP requires HTTPS; HTTP is allowed only for literal loopback test endpoints")
	}
	safeClient := *httpClient
	// net/http does not recognize hibp-api-key as a sensitive redirect header.
	// Reject every redirect, including same-origin redirects, before another request.
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	safeClient.Jar = nil
	return &Client{http: &safeClient, baseURL: strings.TrimRight(u.String(), "/"), key: key}, nil
}
func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, requestError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, &Error{Kind: BadRequest}
	}
	req.Header.Set("hibp-api-key", c.key.Reveal())
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, requestError(ctx, err)
	}
	return resp, nil
}

func (*Client) String() string   { return "hibp.Client" }
func (*Client) GoString() string { return "hibp.Client" }
