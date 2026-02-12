package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewForTest creates a Client with the provided controller-runtime client.
// This is intended for use in tests with fake clients.
func NewForTest(c client.Client) *Client {
	return &Client{client: c}
}

// NewForTestWithDynamic creates a Client with both a controller-runtime client
// and a dynamic client. This is intended for tests that exercise CRD-based handlers.
func NewForTestWithDynamic(c client.Client, dc dynamic.Interface) *Client {
	return &Client{client: c, dynamicClient: dc}
}
