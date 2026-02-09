package kubernetes

import (
	"context"
	"log/slog"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Edition represents the detected NGF edition.
type Edition string

const (
	EditionOSS        Edition = "oss"
	EditionEnterprise Edition = "enterprise"
	EditionUnknown    Edition = "unknown"
)

// enterpriseCRDs are CRDs that only exist in NGF Enterprise.
// Note: snippetsfilters.gateway.nginx.org ships with OSS in v2.4.1+.
var enterpriseCRDs = []string{
	"appolicies.appprotect.f5.com",
	"apdoslogconfs.appprotectdos.f5.com",
}

// DetectEdition checks for enterprise-only CRDs to determine the NGF edition.
func (c *Client) DetectEdition(ctx context.Context) Edition {
	for _, name := range enterpriseCRDs {
		var crd apiextensionsv1.CustomResourceDefinition
		key := client.ObjectKey{Name: name}
		if err := c.client.Get(ctx, key, &crd); err == nil {
			slog.Info("enterprise CRD detected", "crd", name)
			return EditionEnterprise
		}
	}
	return EditionOSS
}
