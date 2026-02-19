package database

import (
	"context"
	"fmt"
	"sync"
)

// MockStore is an in-memory implementation of the Store interface for testing.
type MockStore struct {
	mu         sync.Mutex
	audits     []AuditEntry
	alertRules map[string]AlertRule
	savedViews []SavedView
}

// NewMockStore returns an initialized MockStore.
func NewMockStore() *MockStore {
	return &MockStore{
		audits:     []AuditEntry{},
		alertRules: make(map[string]AlertRule),
		savedViews: []SavedView{},
	}
}

// Migrate is a no-op for the mock store.
func (m *MockStore) Migrate(_ context.Context) error {
	return nil
}

// Close is a no-op for the mock store.
func (m *MockStore) Close() error {
	return nil
}

// InsertAuditEntry appends an audit entry to the in-memory slice.
func (m *MockStore) InsertAuditEntry(_ context.Context, entry AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits = append(m.audits, entry)
	return nil
}

// ListAuditEntries returns audit entries with optional filtering and pagination.
func (m *MockStore) ListAuditEntries(_ context.Context, opts AuditListOptions) ([]AuditEntry, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []AuditEntry
	for _, e := range m.audits {
		if opts.Resource != "" && e.Resource != opts.Resource {
			continue
		}
		if opts.Action != "" && e.Action != opts.Action {
			continue
		}
		if opts.User != "" && e.User != opts.User {
			continue
		}
		if opts.Namespace != "" && e.Namespace != opts.Namespace {
			continue
		}
		if opts.Since != nil && e.Timestamp.Before(*opts.Since) {
			continue
		}
		filtered = append(filtered, e)
	}

	total := int64(len(filtered))

	// Apply pagination
	start := opts.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := len(filtered)
	if opts.Limit > 0 && start+opts.Limit < end {
		end = start + opts.Limit
	}

	result := filtered[start:end]
	return result, total, nil
}

// GetAuditEntry returns the audit entry with the given ID, or nil if not found.
func (m *MockStore) GetAuditEntry(_ context.Context, id string) (*AuditEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.audits {
		if e.ID == id {
			entry := e
			return &entry, nil
		}
	}
	return nil, nil
}

// ListAlertRules returns all alert rules.
func (m *MockStore) ListAlertRules(_ context.Context) ([]AlertRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rules := make([]AlertRule, 0, len(m.alertRules))
	for _, r := range m.alertRules {
		rules = append(rules, r)
	}
	return rules, nil
}

// GetAlertRule returns the alert rule with the given ID, or nil if not found.
func (m *MockStore) GetAlertRule(_ context.Context, id string) (*AlertRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.alertRules[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

// CreateAlertRule stores a new alert rule.
func (m *MockStore) CreateAlertRule(_ context.Context, rule AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.alertRules[rule.ID]; exists {
		return fmt.Errorf("alert rule with id %s already exists", rule.ID)
	}
	m.alertRules[rule.ID] = rule
	return nil
}

// UpdateAlertRule updates an existing alert rule.
func (m *MockStore) UpdateAlertRule(_ context.Context, rule AlertRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.alertRules[rule.ID]; !exists {
		return fmt.Errorf("alert rule with id %s not found", rule.ID)
	}
	m.alertRules[rule.ID] = rule
	return nil
}

// DeleteAlertRule removes an alert rule by ID.
func (m *MockStore) DeleteAlertRule(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.alertRules[id]; !exists {
		return fmt.Errorf("alert rule with id %s not found", id)
	}
	delete(m.alertRules, id)
	return nil
}

// ListSavedViews returns saved views for a given user ID.
func (m *MockStore) ListSavedViews(_ context.Context, userID string) ([]SavedView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var views []SavedView
	for _, v := range m.savedViews {
		if v.UserID == userID {
			views = append(views, v)
		}
	}
	return views, nil
}

// CreateSavedView stores a new saved view.
func (m *MockStore) CreateSavedView(_ context.Context, view SavedView) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedViews = append(m.savedViews, view)
	return nil
}

// DeleteSavedView removes a saved view by ID.
func (m *MockStore) DeleteSavedView(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, v := range m.savedViews {
		if v.ID == id {
			m.savedViews = append(m.savedViews[:i], m.savedViews[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("saved view with id %s not found", id)
}
