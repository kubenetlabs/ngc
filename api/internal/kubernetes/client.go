package kubernetes

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps a controller-runtime client for typed access to Gateway API resources.
type Client struct {
	client client.Client
}

// New creates a new Kubernetes client.
// It tries in-cluster config first, then falls back to the provided kubeconfig path,
// KUBECONFIG env, or ~/.kube/config.
func New(kubeconfig string) (*Client, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding client-go scheme: %w", err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		return nil, fmt.Errorf("adding gateway-api scheme: %w", err)
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding apiextensions scheme: %w", err)
	}

	cfg, err := resolveConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("resolving kubeconfig: %w", err)
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("creating controller-runtime client: %w", err)
	}

	slog.Info("kubernetes client initialized", "host", cfg.Host)
	return &Client{client: c}, nil
}

// NewFromContext creates a new Kubernetes client using the specified kubeconfig path
// and optional context name. This supports multi-cluster configurations where each
// cluster may use a different kubeconfig file and/or context.
func NewFromContext(kubeconfigPath, contextName string) (*Client, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding client-go scheme: %w", err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		return nil, fmt.Errorf("adding gateway-api scheme: %w", err)
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding apiextensions scheme: %w", err)
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig for context %q: %w", contextName, err)
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("creating controller-runtime client: %w", err)
	}

	slog.Info("kubernetes client initialized", "host", cfg.Host, "context", contextName)
	return &Client{client: c}, nil
}

func resolveConfig(kubeconfig string) (*rest.Config, error) {
	// Try in-cluster first.
	if cfg, err := rest.InClusterConfig(); err == nil {
		slog.Info("using in-cluster kubernetes config")
		return cfg, nil
	}

	// Explicit path.
	if kubeconfig != "" {
		slog.Info("using kubeconfig", "path", kubeconfig)
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// KUBECONFIG env.
	if env := os.Getenv("KUBECONFIG"); env != "" {
		slog.Info("using KUBECONFIG env", "path", env)
		return clientcmd.BuildConfigFromFlags("", env)
	}

	// Default ~/.kube/config.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	path := filepath.Join(home, ".kube", "config")
	slog.Info("using default kubeconfig", "path", path)
	return clientcmd.BuildConfigFromFlags("", path)
}
