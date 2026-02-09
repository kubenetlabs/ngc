package kubernetes

import (
	"context"
	"fmt"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListGateways returns all Gateways, optionally filtered by namespace.
func (c *Client) ListGateways(ctx context.Context, namespace string) ([]gatewayv1.Gateway, error) {
	var list gatewayv1.GatewayList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing gateways: %w", err)
	}
	return list.Items, nil
}

// GetGateway returns a single Gateway by namespace and name.
func (c *Client) GetGateway(ctx context.Context, namespace, name string) (*gatewayv1.Gateway, error) {
	var gw gatewayv1.Gateway
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.client.Get(ctx, key, &gw); err != nil {
		return nil, fmt.Errorf("getting gateway %s/%s: %w", namespace, name, err)
	}
	return &gw, nil
}

// ListGatewayClasses returns all GatewayClasses (cluster-scoped).
func (c *Client) ListGatewayClasses(ctx context.Context) ([]gatewayv1.GatewayClass, error) {
	var list gatewayv1.GatewayClassList
	if err := c.client.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("listing gatewayclasses: %w", err)
	}
	return list.Items, nil
}

// GetGatewayClass returns a single GatewayClass by name (cluster-scoped).
func (c *Client) GetGatewayClass(ctx context.Context, name string) (*gatewayv1.GatewayClass, error) {
	var gc gatewayv1.GatewayClass
	key := client.ObjectKey{Name: name}
	if err := c.client.Get(ctx, key, &gc); err != nil {
		return nil, fmt.Errorf("getting gatewayclass %s: %w", name, err)
	}
	return &gc, nil
}

// ListHTTPRoutes returns all HTTPRoutes, optionally filtered by namespace.
func (c *Client) ListHTTPRoutes(ctx context.Context, namespace string) ([]gatewayv1.HTTPRoute, error) {
	var list gatewayv1.HTTPRouteList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing httproutes: %w", err)
	}
	return list.Items, nil
}

// GetHTTPRoute returns a single HTTPRoute by namespace and name.
func (c *Client) GetHTTPRoute(ctx context.Context, namespace, name string) (*gatewayv1.HTTPRoute, error) {
	var hr gatewayv1.HTTPRoute
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.client.Get(ctx, key, &hr); err != nil {
		return nil, fmt.Errorf("getting httproute %s/%s: %w", namespace, name, err)
	}
	return &hr, nil
}
