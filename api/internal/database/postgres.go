package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgres opens a connection to a PostgreSQL database.
func NewPostgres(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return &PostgresStore{db: db}, nil
}

// Migrate creates tables if they don't exist.
func (s *PostgresStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, postgresSchema)
	return err
}

// Close closes the database connection.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// InsertAuditEntry inserts a new audit log entry.
func (s *PostgresStore) InsertAuditEntry(ctx context.Context, entry AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, timestamp, "user", action, resource, name, namespace, cluster, before_json, after_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.ID, entry.Timestamp, entry.User, entry.Action, entry.Resource,
		entry.Name, entry.Namespace, entry.Cluster, entry.BeforeJSON, entry.AfterJSON,
	)
	return err
}

// ListAuditEntries returns paginated audit entries with optional filters.
func (s *PostgresStore) ListAuditEntries(ctx context.Context, opts AuditListOptions) ([]AuditEntry, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if opts.Resource != "" {
		conditions = append(conditions, fmt.Sprintf("resource = $%d", argIdx))
		args = append(args, opts.Resource)
		argIdx++
	}
	if opts.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, opts.Action)
		argIdx++
	}
	if opts.User != "" {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argIdx))
		args = append(args, opts.User)
		argIdx++
	}
	if opts.Namespace != "" {
		conditions = append(conditions, fmt.Sprintf("namespace = $%d", argIdx))
		args = append(args, opts.Namespace)
		argIdx++
	}
	if opts.Since != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *opts.Since)
		argIdx++
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
		`SELECT id, timestamp, "user", action, resource, name, namespace, cluster, before_json, after_json FROM audit_log %s ORDER BY timestamp DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
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
func (s *PostgresStore) GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error) {
	var e AuditEntry
	err := s.db.QueryRowContext(ctx,
		`SELECT id, timestamp, "user", action, resource, name, namespace, cluster, before_json, after_json FROM audit_log WHERE id = $1`,
		id,
	).Scan(&e.ID, &e.Timestamp, &e.User, &e.Action, &e.Resource, &e.Name, &e.Namespace, &e.Cluster, &e.BeforeJSON, &e.AfterJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &e, err
}

// ListAlertRules returns all alert rules.
func (s *PostgresStore) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
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
func (s *PostgresStore) GetAlertRule(ctx context.Context, id string) (*AlertRule, error) {
	var r AlertRule
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, resource, metric, operator, threshold, severity, enabled, created_at, updated_at FROM alert_rules WHERE id = $1",
		id,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Resource, &r.Metric, &r.Operator, &r.Threshold, &r.Severity, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// CreateAlertRule creates a new alert rule.
func (s *PostgresStore) CreateAlertRule(ctx context.Context, rule AlertRule) error {
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO alert_rules (id, name, description, resource, metric, operator, threshold, severity, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		rule.ID, rule.Name, rule.Description, rule.Resource, rule.Metric, rule.Operator, rule.Threshold, rule.Severity, rule.Enabled, now, now,
	)
	return err
}

// UpdateAlertRule updates an existing alert rule.
func (s *PostgresStore) UpdateAlertRule(ctx context.Context, rule AlertRule) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE alert_rules SET name = $1, description = $2, resource = $3, metric = $4, operator = $5, threshold = $6, severity = $7, enabled = $8, updated_at = $9 WHERE id = $10`,
		rule.Name, rule.Description, rule.Resource, rule.Metric, rule.Operator, rule.Threshold, rule.Severity, rule.Enabled, time.Now().UTC(), rule.ID,
	)
	return err
}

// DeleteAlertRule deletes an alert rule by ID.
func (s *PostgresStore) DeleteAlertRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM alert_rules WHERE id = $1", id)
	return err
}

// ListSavedViews returns saved views for a user.
func (s *PostgresStore) ListSavedViews(ctx context.Context, userID string) ([]SavedView, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, user_id, name, view_type, config, created_at FROM saved_views WHERE user_id = $1 ORDER BY name",
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
func (s *PostgresStore) CreateSavedView(ctx context.Context, view SavedView) error {
	if view.ID == "" {
		view.ID = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO saved_views (id, user_id, name, view_type, config, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		view.ID, view.UserID, view.Name, view.ViewType, view.Config, time.Now().UTC(),
	)
	return err
}

// DeleteSavedView deletes a saved view by ID.
func (s *PostgresStore) DeleteSavedView(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM saved_views WHERE id = $1", id)
	return err
}

const postgresSchema = `
CREATE TABLE IF NOT EXISTS audit_log (
	id UUID PRIMARY KEY,
	timestamp TIMESTAMPTZ NOT NULL,
	"user" TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL,
	resource TEXT NOT NULL,
	name TEXT NOT NULL,
	namespace TEXT NOT NULL DEFAULT '',
	cluster TEXT NOT NULL DEFAULT '',
	before_json JSONB NOT NULL DEFAULT '{}',
	after_json JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_log(resource);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);

CREATE TABLE IF NOT EXISTS alert_rules (
	id UUID PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	resource TEXT NOT NULL,
	metric TEXT NOT NULL,
	operator TEXT NOT NULL,
	threshold DOUBLE PRECISION NOT NULL,
	severity TEXT NOT NULL DEFAULT 'warning',
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS saved_views (
	id UUID PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	view_type TEXT NOT NULL,
	config JSONB NOT NULL DEFAULT '{}',
	created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_saved_views_user ON saved_views(user_id);
`
