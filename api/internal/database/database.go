package database

import (
	"context"
	"time"
)

// Store defines the config database interface for audit logs, alert rules, and saved views.
type Store interface {
	// Migrate runs schema migrations.
	Migrate(ctx context.Context) error
	// Close closes the database connection.
	Close() error

	// Audit log
	InsertAuditEntry(ctx context.Context, entry AuditEntry) error
	ListAuditEntries(ctx context.Context, opts AuditListOptions) ([]AuditEntry, int64, error)
	GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error)

	// Alert rules
	ListAlertRules(ctx context.Context) ([]AlertRule, error)
	GetAlertRule(ctx context.Context, id string) (*AlertRule, error)
	CreateAlertRule(ctx context.Context, rule AlertRule) error
	UpdateAlertRule(ctx context.Context, rule AlertRule) error
	DeleteAlertRule(ctx context.Context, id string) error

	// Saved views
	ListSavedViews(ctx context.Context, userID string) ([]SavedView, error)
	CreateSavedView(ctx context.Context, view SavedView) error
	DeleteSavedView(ctx context.Context, id string) error
}

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	User       string    `json:"user"`
	Action     string    `json:"action"`     // create, update, delete
	Resource   string    `json:"resource"`   // e.g., "Gateway", "HTTPRoute", "InferenceStack"
	Name       string    `json:"name"`       // resource name
	Namespace  string    `json:"namespace"`  // resource namespace
	Cluster    string    `json:"cluster"`    // cluster context
	BeforeJSON string    `json:"beforeJson"` // JSON snapshot before change
	AfterJSON  string    `json:"afterJson"`  // JSON snapshot after change
}

// AuditListOptions controls pagination and filtering for audit queries.
type AuditListOptions struct {
	Offset    int
	Limit     int
	Resource  string
	Action    string
	User      string
	Namespace string
	Since     *time.Time
}

// AlertRule defines a threshold-based alert.
type AlertRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`  // e.g., "certificate", "gateway", "inference"
	Metric      string `json:"metric"`    // e.g., "error_rate", "expiry_days", "gpu_util"
	Operator    string `json:"operator"`  // gt, lt, eq
	Threshold   float64 `json:"threshold"`
	Severity    string `json:"severity"` // critical, warning, info
	Enabled     bool   `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// SavedView is a user-specific saved dashboard/filter configuration.
type SavedView struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	ViewType  string    `json:"viewType"` // e.g., "dashboard", "log-query", "metrics"
	Config    string    `json:"config"`   // JSON config
	CreatedAt time.Time `json:"createdAt"`
}
