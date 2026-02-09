package kubernetes

import "sigs.k8s.io/controller-runtime/pkg/client"

// NewForTest creates a Client with the provided controller-runtime client.
// This is intended for use in tests with fake clients.
func NewForTest(c client.Client) *Client {
	return &Client{client: c}
}
