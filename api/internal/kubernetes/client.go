package kubernetes

// Client wraps a Kubernetes client for interacting with the cluster.
//
// This will use client-go and controller-runtime to:
//   - Watch Gateway API resources (GatewayClasses, Gateways, HTTPRoutes, etc.)
//   - Apply/patch/delete Kubernetes resources
//   - Read Pod logs and events
//   - Interact with inference extension CRDs (InferencePool, etc.)
type Client struct {
	// kubeconfig path or in-cluster config will be resolved at construction time.
	// restConfig *rest.Config
	// client     client.Client
}

// New creates a new Kubernetes client.
// It attempts in-cluster configuration first, falling back to kubeconfig.
func New() (*Client, error) {
	// TODO: implement using client-go / controller-runtime
	//
	// config, err := rest.InClusterConfig()
	// if err != nil {
	//     config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	// }
	// ...

	return &Client{}, nil
}
