package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

// Client wraps a ClickHouse database connection.
type Client struct {
	conn ch.Conn
	dsn  string
}

// New creates a new ClickHouse client with the given DSN.
func New(dsn string) (*Client, error) {
	conn, err := ch.Open(&ch.Options{
		Addr: []string{dsn},
		Settings: ch.Settings{
			"max_execution_time": 30,
		},
		DialTimeout: 5 * time.Second,
		Compression: &ch.Compression{
			Method: ch.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}
	if err := conn.Ping(context.Background()); err != nil {
		slog.Warn("clickhouse ping failed, continuing", "error", err)
	}
	slog.Info("clickhouse client connected", "dsn", dsn)
	return &Client{conn: conn, dsn: dsn}, nil
}

// Conn returns the underlying ClickHouse connection for query execution.
func (c *Client) Conn() ch.Conn {
	return c.conn
}

// Ping checks connectivity to ClickHouse.
func (c *Client) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Close closes the ClickHouse connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
