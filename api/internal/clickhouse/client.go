package clickhouse

import "log/slog"

// Client wraps a ClickHouse database connection.
type Client struct {
	// conn clickhouse.Conn
	dsn string
}

// New creates a new ClickHouse client with the given DSN.
func New(dsn string) (*Client, error) {
	// TODO: implement using github.com/ClickHouse/clickhouse-go/v2
	//
	// conn, err := clickhouse.Open(&clickhouse.Options{
	//     Addr: []string{dsn},
	// })

	slog.Info("clickhouse client created (stub)", "dsn", dsn)
	return &Client{dsn: dsn}, nil
}

// Close closes the ClickHouse connection.
func (c *Client) Close() error {
	slog.Info("clickhouse client closed (stub)")
	return nil
}
