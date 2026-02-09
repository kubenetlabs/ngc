package database

import "github.com/kubenetlabs/ngc/api/pkg/types"

// Database defines the interface for persistent storage operations.
// Implementations include ClickHouse for production and in-memory for testing.
type Database interface {
	// Close releases database resources.
	Close() error

	// Audit log operations
	GetAuditLogs(page, pageSize int) ([]types.AuditEntry, int, error)
	CreateAuditLog(entry types.AuditEntry) error

	// Alert rule operations (stubs)
	GetAlertRules() ([]interface{}, error)
	CreateAlertRule(rule interface{}) error
	UpdateAlertRule(id string, rule interface{}) error
	DeleteAlertRule(id string) error

	// Metrics storage operations (stubs)
	StoreMetrics(data interface{}) error
	QueryMetrics(query string, params map[string]interface{}) (interface{}, error)

	// Log storage operations (stubs)
	StoreLogs(data interface{}) error
	QueryLogs(query string, params map[string]interface{}) (interface{}, error)
}
