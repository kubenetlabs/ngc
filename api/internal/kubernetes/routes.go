package kubernetes

import (
	"context"
	"fmt"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// --- GRPCRoute ---

// ListGRPCRoutes returns all GRPCRoutes, optionally filtered by namespace.
func (c *Client) ListGRPCRoutes(ctx context.Context, namespace string) ([]gatewayv1.GRPCRoute, error) {
	var list gatewayv1.GRPCRouteList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing grpcroutes: %w", err)
	}
	return list.Items, nil
}

// GetGRPCRoute returns a single GRPCRoute by namespace and name.
func (c *Client) GetGRPCRoute(ctx context.Context, namespace, name string) (*gatewayv1.GRPCRoute, error) {
	var route gatewayv1.GRPCRoute
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.client.Get(ctx, key, &route); err != nil {
		return nil, fmt.Errorf("getting grpcroute %s/%s: %w", namespace, name, err)
	}
	return &route, nil
}

// CreateGRPCRoute creates a new GRPCRoute and returns the server-populated object.
func (c *Client) CreateGRPCRoute(ctx context.Context, route *gatewayv1.GRPCRoute) (*gatewayv1.GRPCRoute, error) {
	if err := c.client.Create(ctx, route); err != nil {
		return nil, fmt.Errorf("creating grpcroute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// UpdateGRPCRoute updates an existing GRPCRoute and returns the server-populated object.
func (c *Client) UpdateGRPCRoute(ctx context.Context, route *gatewayv1.GRPCRoute) (*gatewayv1.GRPCRoute, error) {
	if err := c.client.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("updating grpcroute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// DeleteGRPCRoute deletes a GRPCRoute by namespace and name.
func (c *Client) DeleteGRPCRoute(ctx context.Context, namespace, name string) error {
	route, err := c.GetGRPCRoute(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("fetching grpcroute for deletion %s/%s: %w", namespace, name, err)
	}
	if err := c.client.Delete(ctx, route); err != nil {
		return fmt.Errorf("deleting grpcroute %s/%s: %w", namespace, name, err)
	}
	return nil
}

// --- TLSRoute ---

// ListTLSRoutes returns all TLSRoutes, optionally filtered by namespace.
func (c *Client) ListTLSRoutes(ctx context.Context, namespace string) ([]gatewayv1alpha2.TLSRoute, error) {
	var list gatewayv1alpha2.TLSRouteList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing tlsroutes: %w", err)
	}
	return list.Items, nil
}

// GetTLSRoute returns a single TLSRoute by namespace and name.
func (c *Client) GetTLSRoute(ctx context.Context, namespace, name string) (*gatewayv1alpha2.TLSRoute, error) {
	var route gatewayv1alpha2.TLSRoute
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.client.Get(ctx, key, &route); err != nil {
		return nil, fmt.Errorf("getting tlsroute %s/%s: %w", namespace, name, err)
	}
	return &route, nil
}

// CreateTLSRoute creates a new TLSRoute and returns the server-populated object.
func (c *Client) CreateTLSRoute(ctx context.Context, route *gatewayv1alpha2.TLSRoute) (*gatewayv1alpha2.TLSRoute, error) {
	if err := c.client.Create(ctx, route); err != nil {
		return nil, fmt.Errorf("creating tlsroute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// UpdateTLSRoute updates an existing TLSRoute and returns the server-populated object.
func (c *Client) UpdateTLSRoute(ctx context.Context, route *gatewayv1alpha2.TLSRoute) (*gatewayv1alpha2.TLSRoute, error) {
	if err := c.client.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("updating tlsroute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// DeleteTLSRoute deletes a TLSRoute by namespace and name.
func (c *Client) DeleteTLSRoute(ctx context.Context, namespace, name string) error {
	route, err := c.GetTLSRoute(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("fetching tlsroute for deletion %s/%s: %w", namespace, name, err)
	}
	if err := c.client.Delete(ctx, route); err != nil {
		return fmt.Errorf("deleting tlsroute %s/%s: %w", namespace, name, err)
	}
	return nil
}

// --- TCPRoute ---

// ListTCPRoutes returns all TCPRoutes, optionally filtered by namespace.
func (c *Client) ListTCPRoutes(ctx context.Context, namespace string) ([]gatewayv1alpha2.TCPRoute, error) {
	var list gatewayv1alpha2.TCPRouteList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing tcproutes: %w", err)
	}
	return list.Items, nil
}

// GetTCPRoute returns a single TCPRoute by namespace and name.
func (c *Client) GetTCPRoute(ctx context.Context, namespace, name string) (*gatewayv1alpha2.TCPRoute, error) {
	var route gatewayv1alpha2.TCPRoute
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.client.Get(ctx, key, &route); err != nil {
		return nil, fmt.Errorf("getting tcproute %s/%s: %w", namespace, name, err)
	}
	return &route, nil
}

// CreateTCPRoute creates a new TCPRoute and returns the server-populated object.
func (c *Client) CreateTCPRoute(ctx context.Context, route *gatewayv1alpha2.TCPRoute) (*gatewayv1alpha2.TCPRoute, error) {
	if err := c.client.Create(ctx, route); err != nil {
		return nil, fmt.Errorf("creating tcproute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// UpdateTCPRoute updates an existing TCPRoute and returns the server-populated object.
func (c *Client) UpdateTCPRoute(ctx context.Context, route *gatewayv1alpha2.TCPRoute) (*gatewayv1alpha2.TCPRoute, error) {
	if err := c.client.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("updating tcproute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// DeleteTCPRoute deletes a TCPRoute by namespace and name.
func (c *Client) DeleteTCPRoute(ctx context.Context, namespace, name string) error {
	route, err := c.GetTCPRoute(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("fetching tcproute for deletion %s/%s: %w", namespace, name, err)
	}
	if err := c.client.Delete(ctx, route); err != nil {
		return fmt.Errorf("deleting tcproute %s/%s: %w", namespace, name, err)
	}
	return nil
}

// --- UDPRoute ---

// ListUDPRoutes returns all UDPRoutes, optionally filtered by namespace.
func (c *Client) ListUDPRoutes(ctx context.Context, namespace string) ([]gatewayv1alpha2.UDPRoute, error) {
	var list gatewayv1alpha2.UDPRouteList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing udproutes: %w", err)
	}
	return list.Items, nil
}

// GetUDPRoute returns a single UDPRoute by namespace and name.
func (c *Client) GetUDPRoute(ctx context.Context, namespace, name string) (*gatewayv1alpha2.UDPRoute, error) {
	var route gatewayv1alpha2.UDPRoute
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.client.Get(ctx, key, &route); err != nil {
		return nil, fmt.Errorf("getting udproute %s/%s: %w", namespace, name, err)
	}
	return &route, nil
}

// CreateUDPRoute creates a new UDPRoute and returns the server-populated object.
func (c *Client) CreateUDPRoute(ctx context.Context, route *gatewayv1alpha2.UDPRoute) (*gatewayv1alpha2.UDPRoute, error) {
	if err := c.client.Create(ctx, route); err != nil {
		return nil, fmt.Errorf("creating udproute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// UpdateUDPRoute updates an existing UDPRoute and returns the server-populated object.
func (c *Client) UpdateUDPRoute(ctx context.Context, route *gatewayv1alpha2.UDPRoute) (*gatewayv1alpha2.UDPRoute, error) {
	if err := c.client.Update(ctx, route); err != nil {
		return nil, fmt.Errorf("updating udproute %s/%s: %w", route.Namespace, route.Name, err)
	}
	return route, nil
}

// DeleteUDPRoute deletes a UDPRoute by namespace and name.
func (c *Client) DeleteUDPRoute(ctx context.Context, namespace, name string) error {
	route, err := c.GetUDPRoute(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("fetching udproute for deletion %s/%s: %w", namespace, name, err)
	}
	if err := c.client.Delete(ctx, route); err != nil {
		return fmt.Errorf("deleting udproute %s/%s: %w", namespace, name, err)
	}
	return nil
}
