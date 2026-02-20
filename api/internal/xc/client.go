package xc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with the F5 Distributed Cloud REST API v1.
type Client struct {
	tenant   string
	apiToken string
	baseURL  string
	http     *http.Client
}

// New creates a new XC API client for the given tenant.
func New(tenant, apiToken string) *Client {
	return &Client{
		tenant:   tenant,
		apiToken: apiToken,
		baseURL:  fmt.Sprintf("https://%s.console.ves.volterra.io/api", tenant),
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Tenant returns the configured tenant name.
func (c *Client) Tenant() string {
	return c.tenant
}

// do executes an HTTP request against the XC API with Bearer token auth.
func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "APIToken "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

// decodeResponse reads and decodes a JSON response body, returning an error for non-2xx status codes.
func decodeResponse[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var xcErr XCError
		if json.Unmarshal(data, &xcErr) == nil && xcErr.Message != "" {
			return nil, fmt.Errorf("XC API error (HTTP %d): %s", resp.StatusCode, xcErr.Message)
		}
		return nil, fmt.Errorf("XC API error (HTTP %d): %s", resp.StatusCode, string(data))
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// TestConnection verifies that the credentials are valid by listing namespaces.
func (c *Client) TestConnection(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/web/namespaces", nil)
	if err != nil {
		return fmt.Errorf("testing connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("authentication failed (HTTP %d): check your API token", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connection test failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// CreateHTTPLoadBalancer creates an HTTP Load Balancer in the given XC namespace.
func (c *Client) CreateHTTPLoadBalancer(ctx context.Context, namespace string, lb HTTPLoadBalancer) (*HTTPLoadBalancer, error) {
	path := fmt.Sprintf("/config/namespaces/%s/http_loadbalancers", namespace)
	resp, err := c.do(ctx, http.MethodPost, path, lb)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP load balancer: %w", err)
	}
	return decodeResponse[HTTPLoadBalancer](resp)
}

// ReplaceHTTPLoadBalancer replaces (updates) an existing HTTP Load Balancer.
func (c *Client) ReplaceHTTPLoadBalancer(ctx context.Context, namespace string, lb HTTPLoadBalancer) (*HTTPLoadBalancer, error) {
	path := fmt.Sprintf("/config/namespaces/%s/http_loadbalancers/%s", namespace, lb.Metadata.Name)
	resp, err := c.do(ctx, http.MethodPut, path, lb)
	if err != nil {
		return nil, fmt.Errorf("replacing HTTP load balancer: %w", err)
	}
	return decodeResponse[HTTPLoadBalancer](resp)
}

// GetHTTPLoadBalancer retrieves an HTTP Load Balancer by name.
func (c *Client) GetHTTPLoadBalancer(ctx context.Context, namespace, name string) (*HTTPLoadBalancer, error) {
	path := fmt.Sprintf("/config/namespaces/%s/http_loadbalancers/%s", namespace, name)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting HTTP load balancer: %w", err)
	}
	return decodeResponse[HTTPLoadBalancer](resp)
}

// GetHTTPLoadBalancerRaw retrieves an HTTP Load Balancer as raw JSON to inspect all fields.
func (c *Client) GetHTTPLoadBalancerRaw(ctx context.Context, namespace, name string) (map[string]any, error) {
	path := fmt.Sprintf("/config/namespaces/%s/http_loadbalancers/%s", namespace, name)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting HTTP load balancer raw: %w", err)
	}
	result, err := decodeResponse[map[string]any](resp)
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// DeleteHTTPLoadBalancer deletes an HTTP Load Balancer by name.
func (c *Client) DeleteHTTPLoadBalancer(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("/config/namespaces/%s/http_loadbalancers/%s", namespace, name)
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting HTTP load balancer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deleting HTTP LB (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// CreateOriginPool creates an origin pool in the given XC namespace.
func (c *Client) CreateOriginPool(ctx context.Context, namespace string, pool OriginPoolConfig) (*OriginPoolConfig, error) {
	path := fmt.Sprintf("/config/namespaces/%s/origin_pools", namespace)
	resp, err := c.do(ctx, http.MethodPost, path, pool)
	if err != nil {
		return nil, fmt.Errorf("creating origin pool: %w", err)
	}
	return decodeResponse[OriginPoolConfig](resp)
}

// ReplaceOriginPool replaces (updates) an existing origin pool.
func (c *Client) ReplaceOriginPool(ctx context.Context, namespace string, pool OriginPoolConfig) (*OriginPoolConfig, error) {
	path := fmt.Sprintf("/config/namespaces/%s/origin_pools/%s", namespace, pool.Metadata.Name)
	resp, err := c.do(ctx, http.MethodPut, path, pool)
	if err != nil {
		return nil, fmt.Errorf("replacing origin pool: %w", err)
	}
	return decodeResponse[OriginPoolConfig](resp)
}

// DeleteOriginPool deletes an origin pool by name.
func (c *Client) DeleteOriginPool(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("/config/namespaces/%s/origin_pools/%s", namespace, name)
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting origin pool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deleting origin pool (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListAppFirewalls returns available WAF policies in the given XC namespace.
func (c *Client) ListAppFirewalls(ctx context.Context, namespace string) ([]AppFirewall, error) {
	path := fmt.Sprintf("/config/namespaces/%s/app_firewalls", namespace)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing app firewalls: %w", err)
	}
	result, err := decodeResponse[XCListResponse[AppFirewall]](resp)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}
