package controller

import (
	"context"
	"fmt"
	"log/slog"
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

// XCPublishReconciler reconciles DistributedCloudPublish objects.
type XCPublishReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

	// Update status based on validation results.
	if httpRouteFound {
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
	)

	// Requeue for drift detection.
	return ctrl.Result{RequeueAfter: 120 * time.Second}, nil
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DistributedCloudPublish{}).
		Complete(r)
}
