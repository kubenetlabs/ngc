package multicluster

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

// ClusterClient holds connection state for a single managed cluster.
type ClusterClient struct {
	mu             sync.RWMutex
	Name           string
	DisplayName    string
	Region         string
	Environment    string
	K8sClient      *kubernetes.Client
	PrometheusURL  string
	IsLocal        bool
	Healthy        bool
	LastHealthCheck time.Time
	K8sVersion     string
	NGFVersion     string
	AgentInstalled bool
	ResourceCounts *ResourceCounts
	GPUCapacity    *GPUCapacitySummary
	CircuitBreaker *CircuitBreaker
}

// SetHealthy updates the health status and last check time (thread-safe).
func (cc *ClusterClient) SetHealthy(healthy bool) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.Healthy = healthy
	cc.LastHealthCheck = time.Now()
}

// SetHeartbeat updates fields from an agent heartbeat (thread-safe).
func (cc *ClusterClient) SetHeartbeat(k8sVersion, ngfVersion string, rc *ResourceCounts, gpu *GPUCapacitySummary) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.K8sVersion = k8sVersion
	cc.NGFVersion = ngfVersion
	cc.AgentInstalled = true
	cc.ResourceCounts = rc
	cc.GPUCapacity = gpu
	cc.Healthy = true
	cc.LastHealthCheck = time.Now()
	cc.CircuitBreaker.RecordSuccess()
}

// ClientPool maintains a thread-safe pool of K8s clients built from
// ManagedCluster CRDs on the hub cluster.
type ClientPool struct {
	mu        sync.RWMutex
	clients   map[string]*ClusterClient
	hubDynamic dynamic.Interface
	namespace  string
}

var managedClusterGVR = schema.GroupVersionResource{
	Group:    "ngf-console.f5.com",
	Version:  "v1alpha1",
	Resource: "managedclusters",
}

// NewClientPool creates a pool that reads ManagedCluster CRDs from the hub.
func NewClientPool(hubDynamic dynamic.Interface, namespace string) *ClientPool {
	return &ClientPool{
		clients:    make(map[string]*ClusterClient),
		hubDynamic: hubDynamic,
		namespace:  namespace,
	}
}

// Sync lists ManagedCluster CRDs and creates/updates/removes clients.
func (p *ClientPool) Sync(ctx context.Context) error {
	list, err := p.hubDynamic.Resource(managedClusterGVR).Namespace(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing ManagedClusters: %w", err)
	}

	desired := make(map[string]bool, len(list.Items))

	for _, item := range list.Items {
		name := item.GetName()
		desired[name] = true

		p.mu.RLock()
		_, exists := p.clients[name]
		p.mu.RUnlock()

		if exists {
			continue
		}

		cc, err := p.buildClient(ctx, &item)
		if err != nil {
			slog.Error("failed to build client for cluster", "cluster", name, "error", err)
			continue
		}

		p.mu.Lock()
		p.clients[name] = cc
		p.mu.Unlock()

		slog.Info("cluster registered", "name", name, "displayName", cc.DisplayName, "region", cc.Region)
	}

	// Remove clients for deleted ManagedClusters.
	p.mu.Lock()
	for name := range p.clients {
		if !desired[name] {
			slog.Info("cluster unregistered", "name", name)
			// K8s clients will be GC'd; controller-runtime doesn't expose transport close.
			delete(p.clients, name)
		}
	}
	p.mu.Unlock()

	return nil
}

// Get returns the ClusterClient by name. Returns an error if the cluster is
// not found or the circuit breaker is open.
func (p *ClientPool) Get(name string) (*ClusterClient, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cc, ok := p.clients[name]
	if !ok {
		return nil, fmt.Errorf("cluster %q not found", name)
	}
	if cc.CircuitBreaker != nil && cc.CircuitBreaker.State() == StateOpen {
		return nil, fmt.Errorf("cluster %q circuit breaker open", name)
	}
	return cc, nil
}

// List returns all registered ClusterClients.
func (p *ClientPool) List() []*ClusterClient {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*ClusterClient, 0, len(p.clients))
	for _, cc := range p.clients {
		result = append(result, cc)
	}
	return result
}

// Names returns the names of all registered clusters.
func (p *ClientPool) Names() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.clients))
	for name := range p.clients {
		names = append(names, name)
	}
	return names
}

// UpdateStatus updates the ManagedCluster status subresource on the hub.
func (p *ClientPool) UpdateStatus(ctx context.Context, name string, status map[string]interface{}) error {
	existing, err := p.hubDynamic.Resource(managedClusterGVR).Namespace(p.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting ManagedCluster %q: %w", name, err)
	}
	existing.Object["status"] = status
	_, err = p.hubDynamic.Resource(managedClusterGVR).Namespace(p.namespace).UpdateStatus(ctx, existing, metav1.UpdateOptions{})
	return err
}

// Namespace returns the namespace used for ManagedCluster CRDs.
func (p *ClientPool) Namespace() string {
	return p.namespace
}

// CreateRaw creates an arbitrary K8s resource via the dynamic client.
func (p *ClientPool) CreateRaw(ctx context.Context, apiVersion, resource string, obj map[string]interface{}) error {
	gvr := schema.GroupVersionResource{Version: apiVersion, Resource: resource}
	u := &unstructured.Unstructured{Object: obj}
	_, err := p.hubDynamic.Resource(gvr).Namespace(p.namespace).Create(ctx, u, metav1.CreateOptions{})
	return err
}

// DeleteRaw deletes an arbitrary K8s resource via the dynamic client.
func (p *ClientPool) DeleteRaw(ctx context.Context, apiVersion, resource, name string) error {
	gvr := schema.GroupVersionResource{Version: apiVersion, Resource: resource}
	return p.hubDynamic.Resource(gvr).Namespace(p.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// CreateManagedCluster creates a ManagedCluster CRD from a raw object.
func (p *ClientPool) CreateManagedCluster(ctx context.Context, obj map[string]interface{}) error {
	u := &unstructured.Unstructured{Object: obj}
	_, err := p.hubDynamic.Resource(managedClusterGVR).Namespace(p.namespace).Create(ctx, u, metav1.CreateOptions{})
	return err
}

// DeleteManagedCluster deletes a ManagedCluster CRD by name.
func (p *ClientPool) DeleteManagedCluster(ctx context.Context, name string) error {
	return p.hubDynamic.Resource(managedClusterGVR).Namespace(p.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// buildClient constructs a ClusterClient from an unstructured ManagedCluster.
func (p *ClientPool) buildClient(ctx context.Context, item *unstructured.Unstructured) (*ClusterClient, error) {
	spec, _ := item.Object["spec"].(map[string]interface{})

	displayName, _ := spec["displayName"].(string)
	region, _ := spec["region"].(string)
	environment, _ := spec["environment"].(string)
	isLocal, _ := spec["isLocal"].(bool)

	cc := &ClusterClient{
		Name:           item.GetName(),
		DisplayName:    displayName,
		Region:         region,
		Environment:    environment,
		IsLocal:        isLocal,
		CircuitBreaker: NewCircuitBreaker(3, 30*time.Second),
	}

	var k8sClient *kubernetes.Client
	var err error

	if isLocal {
		k8sClient, err = kubernetes.New("")
		if err != nil {
			return nil, fmt.Errorf("creating in-cluster client: %w", err)
		}
	} else {
		secretRef, _ := spec["kubeconfigSecretRef"].(map[string]interface{})
		secretName, _ := secretRef["name"].(string)
		if secretName == "" {
			return nil, fmt.Errorf("kubeconfigSecretRef.name is required for non-local clusters")
		}

		kubeconfigData, err := p.readKubeconfigSecret(ctx, secretName)
		if err != nil {
			return nil, fmt.Errorf("reading kubeconfig secret %q: %w", secretName, err)
		}

		k8sClient, err = newClientFromKubeconfigBytes(kubeconfigData)
		if err != nil {
			return nil, fmt.Errorf("creating client from kubeconfig: %w", err)
		}
	}

	cc.K8sClient = k8sClient
	return cc, nil
}

// readKubeconfigSecret reads the kubeconfig data from a Secret.
func (p *ClientPool) readKubeconfigSecret(ctx context.Context, secretName string) ([]byte, error) {
	secretGVR := schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	obj, err := p.hubDynamic.Resource(secretGVR).Namespace(p.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting secret: %w", err)
	}

	data, ok := obj.Object["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("secret has no data field")
	}

	// Try "kubeconfig" key first, then "value"
	for _, key := range []string{"kubeconfig", "value"} {
		if encoded, ok := data[key].(string); ok {
			return []byte(encoded), nil
		}
	}

	return nil, fmt.Errorf("secret %q has no 'kubeconfig' or 'value' key", secretName)
}

// newClientFromKubeconfigBytes creates a kubernetes.Client from raw kubeconfig YAML.
func newClientFromKubeconfigBytes(kubeconfig []byte) (*kubernetes.Client, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("parsing kubeconfig: %w", err)
	}
	return kubernetes.NewFromRestConfig(cfg)
}

