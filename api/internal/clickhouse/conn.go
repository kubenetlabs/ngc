package clickhouse

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Querier abstracts the subset of driver.Conn used by Provider.
// Production code passes a real driver.Conn; tests inject a mock.
type Querier interface {
	Query(ctx context.Context, query string, args ...any) (driver.Rows, error)
	Exec(ctx context.Context, query string, args ...any) error
}
