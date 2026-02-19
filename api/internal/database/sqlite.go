package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite (pure Go, no CGO).
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLite opens or creates a SQLite database at the given path.
// It automatically creates the parent directory if it doesn't exist.
func NewSQLite(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create database directory %s: %w", dir, err)
		}
	}
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	return &SQLiteStore{db: db}, nil
}

// Migrate creates tables if they don't exist.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, sqliteSchema)
	return err
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// InsertAuditEntry inserts a new audit log entry.
func (s *SQLiteStore) InsertAuditEntry(ctx context.Context, entry AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, timestamp, user, action, resource, name, namespace, cluster, before_json, after_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Timestamp, entry.User, entry.Action, entry.Resource,
		entry.Name, entry.Namespace, entry.Cluster, entry.BeforeJSON, entry.AfterJSON,
	)
	return err
}

// ListAuditEntries returns paginated audit entries with optional filters.
func (s *SQLiteStore) ListAuditEntries(ctx context.Context, opts AuditListOptions) ([]AuditEntry, int64, error) {
	var conditions []string
	var args []interface{}

	if opts.Resource != "" {
		conditions = append(conditions, "resource = ?")
		args = append(args, opts.Resource)
	}
	if opts.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, opts.Action)
	}
	if opts.User != "" {
		conditions = append(conditions, "user = ?")
		args = append(args, opts.User)
	}
	if opts.Namespace != "" {
		conditions = append(conditions, "namespace = ?")
		args = append(args, opts.Namespace)
	}
	if opts.Since != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *opts.Since)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := opts.Offset

	query := fmt.Sprintf(
		"SELECT id, timestamp, user, action, resource, name, namespace, cluster, before_json, after_json FROM audit_log %s ORDER BY timestamp DESC LIMIT ? OFFSET ?",
		where,
	)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.User, &e.Action, &e.Resource, &e.Name, &e.Namespace, &e.Cluster, &e.BeforeJSON, &e.AfterJSON); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// GetAuditEntry returns a single audit entry by ID.
func (s *SQLiteStore) GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error) {
	var e AuditEntry
	err := s.db.QueryRowContext(ctx,
		"SELECT id, timestamp, user, action, resource, name, namespace, cluster, before_json, after_json FROM audit_log WHERE id = ?",
		id,
	).Scan(&e.ID, &e.Timestamp, &e.User, &e.Action, &e.Resource, &e.Name, &e.Namespace, &e.Cluster, &e.BeforeJSON, &e.AfterJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &e, err
}

// ListAlertRules returns all alert rules.
func (s *SQLiteStore) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, resource, metric, operator, threshold, severity, enabled, created_at, updated_at FROM alert_rules ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []AlertRule
	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Resource, &r.Metric, &r.Operator, &r.Threshold, &r.Severity, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// GetAlertRule returns a single alert rule by ID.
func (s *SQLiteStore) GetAlertRule(ctx context.Context, id string) (*AlertRule, error) {
	var r AlertRule
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, resource, metric, operator, threshold, severity, enabled, created_at, updated_at FROM alert_rules WHERE id = ?",
		id,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Resource, &r.Metric, &r.Operator, &r.Threshold, &r.Severity, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// CreateAlertRule creates a new alert rule.
func (s *SQLiteStore) CreateAlertRule(ctx context.Context, rule AlertRule) error {
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO alert_rules (id, name, description, resource, metric, operator, threshold, severity, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.Name, rule.Description, rule.Resource, rule.Metric, rule.Operator, rule.Threshold, rule.Severity, rule.Enabled, now, now,
	)
	return err
}

// UpdateAlertRule updates an existing alert rule.
func (s *SQLiteStore) UpdateAlertRule(ctx context.Context, rule AlertRule) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE alert_rules SET name = ?, description = ?, resource = ?, metric = ?, operator = ?, threshold = ?, severity = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		rule.Name, rule.Description, rule.Resource, rule.Metric, rule.Operator, rule.Threshold, rule.Severity, rule.Enabled, time.Now().UTC(), rule.ID,
	)
	return err
}

// DeleteAlertRule deletes an alert rule by ID.
func (s *SQLiteStore) DeleteAlertRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM alert_rules WHERE id = ?", id)
	return err
}

// ListSavedViews returns saved views for a user.
func (s *SQLiteStore) ListSavedViews(ctx context.Context, userID string) ([]SavedView, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, user_id, name, view_type, config, created_at FROM saved_views WHERE user_id = ? ORDER BY name",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []SavedView
	for rows.Next() {
		var v SavedView
		if err := rows.Scan(&v.ID, &v.UserID, &v.Name, &v.ViewType, &v.Config, &v.CreatedAt); err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, rows.Err()
}

// CreateSavedView creates a new saved view.
func (s *SQLiteStore) CreateSavedView(ctx context.Context, view SavedView) error {
	if view.ID == "" {
		view.ID = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO saved_views (id, user_id, name, view_type, config, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		view.ID, view.UserID, view.Name, view.ViewType, view.Config, time.Now().UTC(),
	)
	return err
}

// DeleteSavedView deletes a saved view by ID.
func (s *SQLiteStore) DeleteSavedView(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM saved_views WHERE id = ?", id)
	return err
}

// GetXCCredentials returns the stored XC credentials (there is at most one row).
func (s *SQLiteStore) GetXCCredentials(ctx context.Context) (*XCCredentials, error) {
	var c XCCredentials
	err := s.db.QueryRowContext(ctx,
		"SELECT id, tenant, api_token, namespace, created_at, updated_at FROM xc_credentials LIMIT 1",
	).Scan(&c.ID, &c.Tenant, &c.APIToken, &c.Namespace, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// SaveXCCredentials upserts XC credentials (replaces any existing row).
func (s *SQLiteStore) SaveXCCredentials(ctx context.Context, creds XCCredentials) error {
	if creds.ID == "" {
		creds.ID = uuid.NewString()
	}
	now := time.Now().UTC()

	// Delete any existing row then insert (upsert).
	_, _ = s.db.ExecContext(ctx, "DELETE FROM xc_credentials")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO xc_credentials (id, tenant, api_token, namespace, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		creds.ID, creds.Tenant, creds.APIToken, creds.Namespace, now, now,
	)
	return err
}

// DeleteXCCredentials removes stored XC credentials.
func (s *SQLiteStore) DeleteXCCredentials(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM xc_credentials")
	return err
}

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS audit_log (
	id TEXT PRIMARY KEY,
	timestamp DATETIME NOT NULL,
	user TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL,
	resource TEXT NOT NULL,
	name TEXT NOT NULL,
	namespace TEXT NOT NULL DEFAULT '',
	cluster TEXT NOT NULL DEFAULT '',
	before_json TEXT NOT NULL DEFAULT '',
	after_json TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_log(resource);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);

CREATE TABLE IF NOT EXISTS alert_rules (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	resource TEXT NOT NULL,
	metric TEXT NOT NULL,
	operator TEXT NOT NULL,
	threshold REAL NOT NULL,
	severity TEXT NOT NULL DEFAULT 'warning',
	enabled BOOLEAN NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS saved_views (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	view_type TEXT NOT NULL,
	config TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_saved_views_user ON saved_views(user_id);

CREATE INDEX IF NOT EXISTS idx_audit_cluster ON audit_log(cluster);

CREATE TABLE IF NOT EXISTS managed_clusters (
	name TEXT PRIMARY KEY,
	display_name TEXT NOT NULL,
	region TEXT NOT NULL DEFAULT '',
	environment TEXT NOT NULL DEFAULT '',
	phase TEXT NOT NULL DEFAULT 'Pending',
	kubernetes_version TEXT NOT NULL DEFAULT '',
	ngf_version TEXT NOT NULL DEFAULT '',
	agent_installed BOOLEAN NOT NULL DEFAULT 0,
	last_heartbeat DATETIME,
	total_gpus INTEGER NOT NULL DEFAULT 0,
	registered_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS cluster_groups (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	cluster_names TEXT NOT NULL DEFAULT '[]',
	created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS xc_credentials (
	id TEXT PRIMARY KEY,
	tenant TEXT NOT NULL,
	api_token TEXT NOT NULL,
	namespace TEXT NOT NULL DEFAULT 'default',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
`
