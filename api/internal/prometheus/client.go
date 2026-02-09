package prometheus

import "log/slog"

// Client wraps a Prometheus HTTP API client.
type Client struct {
	// api v1.API
	url string
}

// New creates a new Prometheus client pointing at the given URL.
func New(url string) (*Client, error) {
	// TODO: implement using github.com/prometheus/client_golang/api
	//
	// client, err := api.NewClient(api.Config{Address: url})
	// v1api := v1.NewAPI(client)

	slog.Info("prometheus client created (stub)", "url", url)
	return &Client{url: url}, nil
}

// Query executes a PromQL instant query and returns the result.
func (c *Client) Query(promql string) (interface{}, error) {
	// TODO: implement using v1.API.Query()
	slog.Info("prometheus query (stub)", "query", promql)
	return nil, nil
}

// QueryRange executes a PromQL range query and returns the result.
func (c *Client) QueryRange(promql string, start, end string, step string) (interface{}, error) {
	// TODO: implement using v1.API.QueryRange()
	slog.Info("prometheus range query (stub)", "query", promql, "start", start, "end", end, "step", step)
	return nil, nil
}
