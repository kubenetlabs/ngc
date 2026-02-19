package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

// httpRouteGVK is already defined as a function in inferencestack_children.go,
// so the XC controller reuses it via httpRouteGVK() calls.

// xcAPIClient is a minimal XC API client for the operator controller.
type xcAPIClient struct {
	tenant   string
	apiToken string
	baseURL  string
	http     *http.Client
}

func newXCAPIClient(tenant, apiToken string) *xcAPIClient {
	return &xcAPIClient{
		tenant:   tenant,
		apiToken: apiToken,
		baseURL:  fmt.Sprintf("https://%s.console.ves.volterra.io/api", tenant),
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *xcAPIClient) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "APIToken "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.http.Do(req)
}

func (c *xcAPIClient) getHTTPLoadBalancer(ctx context.Context, namespace, name string) (map[string]any, error) {
	path := fmt.Sprintf("/config/namespaces/%s/http_loadbalancers/%s", namespace, name)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *xcAPIClient) deleteHTTPLoadBalancer(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("/config/namespaces/%s/http_loadbalancers/%s", namespace, name)
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *xcAPIClient) deleteOriginPool(ctx context.Context, namespace, name string) error {
	path := fmt.Sprintf("/config/namespaces/%s/origin_pools/%s", namespace, name)
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// XCPublishReconciler reconciles DistributedCloudPublish objects.
type XCPublishReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	xcClient *xcAPIClient
}

// Reconcile handles reconciliation of DistributedCloudPublish resources.
func (r *XCPublishReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := slog.With("controller", "XCPublish", "name", req.Name, "namespace", req.Namespace)

	var publish v1alpha1.DistributedCloudPublish
	if err := r.Get(ctx, req.NamespacedName, &publish); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("reconciling DistributedCloudPublish",
		"httpRouteRef", publish.Spec.HTTPRouteRef,
		"tenant", publish.Spec.DistributedCloud.Tenant,
	)

	// Handle deletion via finalizer.
	if !publish.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&publish, v1alpha1.DistributedCloudPublishFinalizer) {
			log.Info("handling deletion, cleaning up XC resources")

			// Clean up XC resources if we have a client.
			r.cleanupXCResources(ctx, &publish)

			// Remove the finalizer to allow Kubernetes to delete the resource.
			controllerutil.RemoveFinalizer(&publish, v1alpha1.DistributedCloudPublishFinalizer)
			if err := r.Update(ctx, &publish); err != nil {
				return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present.
	if !controllerutil.ContainsFinalizer(&publish, v1alpha1.DistributedCloudPublishFinalizer) {
		controllerutil.AddFinalizer(&publish, v1alpha1.DistributedCloudPublishFinalizer)
		if err := r.Update(ctx, &publish); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		// Re-fetch after update to avoid working with stale resource version.
		if err := r.Get(ctx, req.NamespacedName, &publish); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// Validate that the referenced HTTPRoute exists.
	httpRouteFound := r.httpRouteExists(ctx, publish.Namespace, publish.Spec.HTTPRouteRef)

	// Check XC resource status if client is available.
	xcSynced := false
	if r.xcClient != nil && httpRouteFound {
		xcNs := publish.Spec.DistributedCloud.Namespace
		if xcNs == "" {
			xcNs = "default"
		}
		lbName := "ngf-" + publish.Spec.HTTPRouteRef

		lb, err := r.xcClient.getHTTPLoadBalancer(ctx, xcNs, lbName)
		if err != nil {
			log.Warn("failed to check XC HTTP LB", "name", lbName, "error", err)
		} else if lb != nil {
			xcSynced = true
			publish.Status.XCLoadBalancerName = lbName
			publish.Status.XCOriginPoolName = lbName + "-pool"
			now := metav1.Now()
			publish.Status.LastSyncedAt = &now

			if publish.Spec.DistributedCloud.WAFPolicy != "" {
				publish.Status.WAFPolicyAttached = publish.Spec.DistributedCloud.WAFPolicy
			}
		} else {
			// LB doesn't exist in XC — mark as drift.
			log.Warn("XC HTTP LB not found, possible drift", "name", lbName)
		}
	}

	// Update status based on validation results.
	if httpRouteFound && xcSynced {
		publish.Status.Phase = "Published"
		v1alpha1.SetCondition(&publish.Status.Conditions, v1alpha1.ConditionReady,
			metav1.ConditionTrue, "Published",
			fmt.Sprintf("HTTPRoute %q found and XC resources are synced", publish.Spec.HTTPRouteRef))
	} else if httpRouteFound && r.xcClient != nil {
		publish.Status.Phase = v1alpha1.PhaseDegraded
		v1alpha1.SetCondition(&publish.Status.Conditions, v1alpha1.ConditionReady,
			metav1.ConditionFalse, "XCResourceMissing",
			fmt.Sprintf("HTTPRoute %q found but XC resources may not be in sync", publish.Spec.HTTPRouteRef))
	} else if httpRouteFound {
		publish.Status.Phase = "Published"
		v1alpha1.SetCondition(&publish.Status.Conditions, v1alpha1.ConditionReady,
			metav1.ConditionTrue, "Published",
			fmt.Sprintf("HTTPRoute %q found and configuration is valid", publish.Spec.HTTPRouteRef))
	} else {
		publish.Status.Phase = v1alpha1.PhasePending
		v1alpha1.SetCondition(&publish.Status.Conditions, v1alpha1.ConditionReady,
			metav1.ConditionFalse, "HTTPRouteNotFound",
			fmt.Sprintf("HTTPRoute %q not found in namespace %q", publish.Spec.HTTPRouteRef, publish.Namespace))
	}

	if err := r.Status().Update(ctx, &publish); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status: %w", err)
	}

	log.Info("reconciliation complete",
		"phase", publish.Status.Phase,
		"httpRouteFound", httpRouteFound,
		"xcSynced", xcSynced,
	)

	// Requeue for drift detection.
	return ctrl.Result{RequeueAfter: 120 * time.Second}, nil
}

// cleanupXCResources deletes XC HTTP LB and origin pool for a publish being deleted.
func (r *XCPublishReconciler) cleanupXCResources(ctx context.Context, publish *v1alpha1.DistributedCloudPublish) {
	if r.xcClient == nil {
		return
	}

	xcNs := publish.Spec.DistributedCloud.Namespace
	if xcNs == "" {
		xcNs = "default"
	}

	// Delete HTTP LB.
	lbName := publish.Status.XCLoadBalancerName
	if lbName == "" {
		lbName = "ngf-" + publish.Spec.HTTPRouteRef
	}
	if err := r.xcClient.deleteHTTPLoadBalancer(ctx, xcNs, lbName); err != nil {
		slog.Warn("failed to delete XC HTTP LB on cleanup", "name", lbName, "error", err)
	} else {
		slog.Info("deleted XC HTTP LB", "name", lbName)
	}

	// Delete origin pool.
	poolName := publish.Status.XCOriginPoolName
	if poolName == "" {
		poolName = "ngf-" + publish.Spec.HTTPRouteRef + "-pool"
	}
	if err := r.xcClient.deleteOriginPool(ctx, xcNs, poolName); err != nil {
		slog.Warn("failed to delete XC origin pool on cleanup", "name", poolName, "error", err)
	} else {
		slog.Info("deleted XC origin pool", "name", poolName)
	}
}

// httpRouteExists checks whether an HTTPRoute with the given name exists in the specified namespace
// using an unstructured client lookup.
func (r *XCPublishReconciler) httpRouteExists(ctx context.Context, namespace, name string) bool {
	route := &unstructured.Unstructured{}
	route.SetGroupVersionKind(httpRouteGVK())

	err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, route)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		slog.Warn("error checking HTTPRoute existence", "name", name, "namespace", namespace, "error", err)
		return false
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *XCPublishReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize XC client from environment if configured.
	xcTenant := os.Getenv("XC_TENANT")
	xcToken := os.Getenv("XC_API_TOKEN")
	if xcTenant != "" && xcToken != "" {
		r.xcClient = newXCAPIClient(xcTenant, xcToken)
		slog.Info("XC API client configured for operator", "tenant", xcTenant)
	} else {
		slog.Info("XC API client not configured (XC_TENANT/XC_API_TOKEN not set)")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DistributedCloudPublish{}).
		Complete(r)
}
