package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	store, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewSQLite_CreatesDirAndDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.db")
	store, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(filepath.Join(dir, "subdir")); os.IsNotExist(err) {
		t.Error("expected subdir to be created")
	}
}

func TestMigrate_Succeeds(t *testing.T) {
	store := newTestDB(t)
	// Running migrate again should be idempotent.
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestAuditEntry_InsertAndGet(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	entry := AuditEntry{
		ID:        "audit-1",
		Timestamp: time.Now().UTC().Truncate(time.Second),
		User:      "admin",
		Action:    "create",
		Resource:  "Gateway",
		Name:      "my-gw",
		Namespace: "default",
		Cluster:   "prod",
	}

	if err := store.InsertAuditEntry(ctx, entry); err != nil {
		t.Fatalf("InsertAuditEntry: %v", err)
	}

	got, err := store.GetAuditEntry(ctx, "audit-1")
	if err != nil {
		t.Fatalf("GetAuditEntry: %v", err)
	}
	if got == nil {
		t.Fatal("GetAuditEntry returned nil")
	}
	if got.ID != "audit-1" {
		t.Errorf("ID = %q, want %q", got.ID, "audit-1")
	}
	if got.User != "admin" {
		t.Errorf("User = %q, want %q", got.User, "admin")
	}
	if got.Resource != "Gateway" {
		t.Errorf("Resource = %q, want %q", got.Resource, "Gateway")
	}
}

func TestAuditEntry_GetNonExistent(t *testing.T) {
	store := newTestDB(t)
	got, err := store.GetAuditEntry(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("GetAuditEntry: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent entry, got %v", got)
	}
}

func TestAuditEntry_AutoID(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	entry := AuditEntry{
		Action:   "delete",
		Resource: "HTTPRoute",
		Name:     "test-route",
	}
	if err := store.InsertAuditEntry(ctx, entry); err != nil {
		t.Fatalf("InsertAuditEntry: %v", err)
	}

	entries, total, err := store.ListAuditEntries(ctx, AuditListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditEntries: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if entries[0].ID == "" {
		t.Error("expected auto-generated ID, got empty")
	}
}

func TestListAuditEntries_Filters(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	entries := []AuditEntry{
		{ID: "a1", Action: "create", Resource: "Gateway", Namespace: "default", Timestamp: time.Now().UTC()},
		{ID: "a2", Action: "delete", Resource: "HTTPRoute", Namespace: "default", Timestamp: time.Now().UTC()},
		{ID: "a3", Action: "create", Resource: "Gateway", Namespace: "kube-system", Timestamp: time.Now().UTC()},
	}
	for _, e := range entries {
		if err := store.InsertAuditEntry(ctx, e); err != nil {
			t.Fatalf("InsertAuditEntry: %v", err)
		}
	}

	// Filter by resource
	results, total, err := store.ListAuditEntries(ctx, AuditListOptions{Resource: "Gateway", Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditEntries (resource): %v", err)
	}
	if total != 2 {
		t.Errorf("resource filter: total = %d, want 2", total)
	}

	// Filter by action
	results, total, err = store.ListAuditEntries(ctx, AuditListOptions{Action: "delete", Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditEntries (action): %v", err)
	}
	if total != 1 {
		t.Errorf("action filter: total = %d, want 1", total)
	}
	if results[0].ID != "a2" {
		t.Errorf("action filter: ID = %q, want %q", results[0].ID, "a2")
	}

	// Filter by namespace
	results, total, err = store.ListAuditEntries(ctx, AuditListOptions{Namespace: "kube-system", Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditEntries (namespace): %v", err)
	}
	if total != 1 {
		t.Errorf("namespace filter: total = %d, want 1", total)
	}
	_ = results
}

func TestListAuditEntries_Pagination(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		e := AuditEntry{
			Action:    "create",
			Resource:  "Pod",
			Name:      "pod-" + string(rune('a'+i)),
			Timestamp: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		if err := store.InsertAuditEntry(ctx, e); err != nil {
			t.Fatalf("InsertAuditEntry: %v", err)
		}
	}

	// Page 1: limit=2, offset=0
	results, total, err := store.ListAuditEntries(ctx, AuditListOptions{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("ListAuditEntries page1: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(results) != 2 {
		t.Errorf("page1 len = %d, want 2", len(results))
	}

	// Page 2: limit=2, offset=2
	results, _, err = store.ListAuditEntries(ctx, AuditListOptions{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("ListAuditEntries page2: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("page2 len = %d, want 2", len(results))
	}

	// Page 3: limit=2, offset=4
	results, _, err = store.ListAuditEntries(ctx, AuditListOptions{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("ListAuditEntries page3: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("page3 len = %d, want 1", len(results))
	}
}

func TestAlertRule_CRUD(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	rule := AlertRule{
		ID:        "rule-1",
		Name:      "High Error Rate",
		Resource:  "gateway",
		Metric:    "error_rate",
		Operator:  "gt",
		Threshold: 5.0,
		Severity:  "critical",
		Enabled:   true,
	}

	// Create
	if err := store.CreateAlertRule(ctx, rule); err != nil {
		t.Fatalf("CreateAlertRule: %v", err)
	}

	// Get
	got, err := store.GetAlertRule(ctx, "rule-1")
	if err != nil {
		t.Fatalf("GetAlertRule: %v", err)
	}
	if got == nil {
		t.Fatal("GetAlertRule returned nil")
	}
	if got.Name != "High Error Rate" {
		t.Errorf("Name = %q, want %q", got.Name, "High Error Rate")
	}
	if got.Threshold != 5.0 {
		t.Errorf("Threshold = %f, want 5.0", got.Threshold)
	}
	if !got.Enabled {
		t.Error("expected Enabled=true")
	}

	// List
	rules, err := store.ListAlertRules(ctx)
	if err != nil {
		t.Fatalf("ListAlertRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("ListAlertRules returned %d rules, want 1", len(rules))
	}

	// Update
	rule.Name = "Updated Name"
	rule.Enabled = false
	if err := store.UpdateAlertRule(ctx, rule); err != nil {
		t.Fatalf("UpdateAlertRule: %v", err)
	}
	got, _ = store.GetAlertRule(ctx, "rule-1")
	if got.Name != "Updated Name" {
		t.Errorf("after update: Name = %q, want %q", got.Name, "Updated Name")
	}
	if got.Enabled {
		t.Error("after update: expected Enabled=false")
	}

	// Delete
	if err := store.DeleteAlertRule(ctx, "rule-1"); err != nil {
		t.Fatalf("DeleteAlertRule: %v", err)
	}
	got, _ = store.GetAlertRule(ctx, "rule-1")
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestAlertRule_GetNonExistent(t *testing.T) {
	store := newTestDB(t)
	got, err := store.GetAlertRule(context.Background(), "nope")
	if err != nil {
		t.Fatalf("GetAlertRule: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestSavedView_CRUD(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	view := SavedView{
		ID:       "view-1",
		UserID:   "user-alice",
		Name:     "My Dashboard",
		ViewType: "dashboard",
		Config:   `{"filters":["namespace=default"]}`,
	}

	// Create
	if err := store.CreateSavedView(ctx, view); err != nil {
		t.Fatalf("CreateSavedView: %v", err)
	}

	// Create another view for a different user
	view2 := SavedView{
		ID:       "view-2",
		UserID:   "user-bob",
		Name:     "Bob's View",
		ViewType: "log-query",
		Config:   `{}`,
	}
	if err := store.CreateSavedView(ctx, view2); err != nil {
		t.Fatalf("CreateSavedView: %v", err)
	}

	// List by user
	views, err := store.ListSavedViews(ctx, "user-alice")
	if err != nil {
		t.Fatalf("ListSavedViews: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("ListSavedViews for alice: len = %d, want 1", len(views))
	}
	if views[0].Name != "My Dashboard" {
		t.Errorf("Name = %q, want %q", views[0].Name, "My Dashboard")
	}

	// Delete
	if err := store.DeleteSavedView(ctx, "view-1"); err != nil {
		t.Fatalf("DeleteSavedView: %v", err)
	}
	views, _ = store.ListSavedViews(ctx, "user-alice")
	if len(views) != 0 {
		t.Errorf("after delete: len = %d, want 0", len(views))
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "close-test.db")
	store, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
