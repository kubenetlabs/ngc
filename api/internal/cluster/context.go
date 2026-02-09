package cluster

import (
	"context"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

type contextKey int

const (
	clientKey      contextKey = iota
	clusterNameKey
)

// WithClient stores a Kubernetes client in the context.
func WithClient(ctx context.Context, c *kubernetes.Client) context.Context {
	return context.WithValue(ctx, clientKey, c)
}

// ClientFromContext retrieves the Kubernetes client from the context.
func ClientFromContext(ctx context.Context) *kubernetes.Client {
	c, _ := ctx.Value(clientKey).(*kubernetes.Client)
	return c
}

// WithClusterName stores the active cluster name in the context.
func WithClusterName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, clusterNameKey, name)
}

// ClusterNameFromContext retrieves the active cluster name from the context.
func ClusterNameFromContext(ctx context.Context) string {
	name, _ := ctx.Value(clusterNameKey).(string)
	return name
}
