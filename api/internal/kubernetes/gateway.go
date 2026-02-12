package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
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

// CreateGateway creates a new Gateway and returns the server-populated object.
func (c *Client) CreateGateway(ctx context.Context, gw *gatewayv1.Gateway) (*gatewayv1.Gateway, error) {
	if err := c.client.Create(ctx, gw); err != nil {
		return nil, fmt.Errorf("creating gateway %s/%s: %w", gw.Namespace, gw.Name, err)
	}
	return gw, nil
}

// UpdateGateway updates an existing Gateway and returns the server-populated object.
func (c *Client) UpdateGateway(ctx context.Context, gw *gatewayv1.Gateway) (*gatewayv1.Gateway, error) {
	if err := c.client.Update(ctx, gw); err != nil {
		return nil, fmt.Errorf("updating gateway %s/%s: %w", gw.Namespace, gw.Name, err)
	}
	return gw, nil
}

// DeleteGateway deletes a Gateway by namespace and name.
func (c *Client) DeleteGateway(ctx context.Context, namespace, name string) error {
	gw, err := c.GetGateway(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("fetching gateway for deletion %s/%s: %w", namespace, name, err)
	}
	if err := c.client.Delete(ctx, gw); err != nil {
		return fmt.Errorf("deleting gateway %s/%s: %w", namespace, name, err)
	}
	return nil
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

// CreateHTTPRoute creates a new HTTPRoute and returns the server-populated object.
func (c *Client) CreateHTTPRoute(ctx context.Context, hr *gatewayv1.HTTPRoute) (*gatewayv1.HTTPRoute, error) {
	if err := c.client.Create(ctx, hr); err != nil {
		return nil, fmt.Errorf("creating httproute %s/%s: %w", hr.Namespace, hr.Name, err)
	}
	return hr, nil
}

// UpdateHTTPRoute updates an existing HTTPRoute and returns the server-populated object.
func (c *Client) UpdateHTTPRoute(ctx context.Context, hr *gatewayv1.HTTPRoute) (*gatewayv1.HTTPRoute, error) {
	if err := c.client.Update(ctx, hr); err != nil {
		return nil, fmt.Errorf("updating httproute %s/%s: %w", hr.Namespace, hr.Name, err)
	}
	return hr, nil
}

// DeleteHTTPRoute deletes an HTTPRoute by namespace and name.
func (c *Client) DeleteHTTPRoute(ctx context.Context, namespace, name string) error {
	hr, err := c.GetHTTPRoute(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("fetching httproute for deletion %s/%s: %w", namespace, name, err)
	}
	if err := c.client.Delete(ctx, hr); err != nil {
		return fmt.Errorf("deleting httproute %s/%s: %w", namespace, name, err)
	}
	return nil
}

// ListServices returns all Services, optionally filtered by namespace.
func (c *Client) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	var list corev1.ServiceList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	return list.Items, nil
}

// ListSecrets returns all Secrets, optionally filtered by namespace.
func (c *Client) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	var list corev1.SecretList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := c.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("listing secrets: %w", err)
	}
	return list.Items, nil
}

// GetSecret returns a single Secret by namespace and name.
func (c *Client) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	var secret corev1.Secret
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.client.Get(ctx, key, &secret); err != nil {
		return nil, fmt.Errorf("getting secret %s/%s: %w", namespace, name, err)
	}
	return &secret, nil
}

// DeleteSecret deletes a Secret by namespace and name.
func (c *Client) DeleteSecret(ctx context.Context, namespace, name string) error {
	secret, err := c.GetSecret(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("fetching secret for deletion %s/%s: %w", namespace, name, err)
	}
	if err := c.client.Delete(ctx, secret); err != nil {
		return fmt.Errorf("deleting secret %s/%s: %w", namespace, name, err)
	}
	return nil
}
