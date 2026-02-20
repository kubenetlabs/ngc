package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kubenetlabs/ngc/api/internal/database"
)

// auditLog records an audit entry for a CRUD operation. It is fire-and-forget:
// errors are logged but never returned to the caller, so audit logging cannot
// break the primary request flow.
func auditLog(store database.Store, ctx context.Context, action, resource, name, namespace string, before, after any) {
	if store == nil {
		return
	}

	var beforeJSON, afterJSON string
	if before != nil {
		if b, err := json.Marshal(before); err == nil {
			beforeJSON = string(b)
		}
	}
	if after != nil {
		if b, err := json.Marshal(after); err == nil {
			afterJSON = string(b)
		}
	}

	entry := database.AuditEntry{
		ID:         uuid.NewString(),
		Timestamp:  time.Now().UTC(),
		Action:     action,
		Resource:   resource,
		Name:       name,
		Namespace:  namespace,
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
	}

	if err := store.InsertAuditEntry(ctx, entry); err != nil {
		slog.Error("failed to insert audit entry", "error", err, "action", action, "resource", resource, "name", name)
	}
}
