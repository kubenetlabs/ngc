package kubernetes

import (
	"context"
	"log/slog"
	"sync"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Edition represents the detected NGF edition.
type Edition string

const (
	EditionOSS        Edition = "oss"
	EditionEnterprise Edition = "enterprise"
	EditionUnknown    Edition = "unknown"
)

// editionCacheTTL controls how long a cached edition result is considered valid.
// Enterprise results are cached longer because they are definitive (CRD exists).
// OSS results use a shorter TTL so we re-check in case CRDs were just installed.
const (
	editionEnterpriseTTL = 5 * time.Minute
	editionOSSTTL        = 30 * time.Second
)

// enterpriseCRDs are CRDs that only exist in NGF Enterprise.
// Note: snippetsfilters.gateway.nginx.org ships with OSS in v2.4.1+.
var enterpriseCRDs = []string{
	"appolicies.appprotect.f5.com",
	"apdoslogconfs.appprotectdos.f5.com",
}

// editionCache stores the cached edition detection result per Client.
type editionCache struct {
	mu      sync.RWMutex
	edition Edition
	expires time.Time
}

// DetectEdition checks for enterprise-only CRDs to determine the NGF edition.
// Results are cached to avoid per-request CRD lookups. Transient errors
// (anything other than NotFound) return EditionUnknown instead of EditionOSS
// to prevent false downgrades during pod startup or network blips.
func (c *Client) DetectEdition(ctx context.Context) Edition {
	// Check cache first.
	c.edition.mu.RLock()
	if c.edition.edition != "" && time.Now().Before(c.edition.expires) {
		cached := c.edition.edition
		c.edition.mu.RUnlock()
		return cached
	}
	c.edition.mu.RUnlock()

	// Perform detection.
	result := c.detectEditionUncached(ctx)

	// Cache the result.
	c.edition.mu.Lock()
	c.edition.edition = result
	if result == EditionEnterprise {
		c.edition.expires = time.Now().Add(editionEnterpriseTTL)
	} else if result == EditionOSS {
		c.edition.expires = time.Now().Add(editionOSSTTL)
	} else {
		// Unknown (transient error) — retry quickly.
		c.edition.expires = time.Now().Add(5 * time.Second)
	}
	c.edition.mu.Unlock()

	return result
}

// detectEditionUncached performs the actual CRD lookup.
func (c *Client) detectEditionUncached(ctx context.Context) Edition {
	for _, name := range enterpriseCRDs {
		var crd apiextensionsv1.CustomResourceDefinition
		key := client.ObjectKey{Name: name}
		err := c.client.Get(ctx, key, &crd)
		if err == nil {
			slog.Info("enterprise CRD detected", "crd", name)
			return EditionEnterprise
		}
		// NotFound means the CRD genuinely doesn't exist — this is evidence of OSS.
		// Any other error (network timeout, RBAC not ready, etc.) is transient.
		if !apierrors.IsNotFound(err) {
			slog.Warn("transient error during edition detection, returning unknown",
				"crd", name, "error", err)
			return EditionUnknown
		}
		// NotFound — continue checking the next CRD.
		slog.Debug("enterprise CRD not found", "crd", name)
	}
	// All CRDs returned NotFound — this is genuinely OSS.
	return EditionOSS
}
