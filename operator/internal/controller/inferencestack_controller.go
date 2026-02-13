package controller

import (
	"context"
	"log/slog"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

const (
	reconcileInterval = 60 * time.Second
)

// InferenceStackReconciler reconciles InferenceStack objects.
type InferenceStackReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles reconciliation of InferenceStack resources.
func (r *InferenceStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := slog.With("controller", "InferenceStack", "name", req.Name, "namespace", req.Namespace)

	// 1. Fetch parent CRD
	var stack v1alpha1.InferenceStack
	if err := r.Get(ctx, req.NamespacedName, &stack); err != nil {
		if errors.IsNotFound(err) {
			log.Info("InferenceStack not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error("unable to fetch InferenceStack", "error", err)
		return ctrl.Result{}, err
	}

	// 2. Handle deletion via finalizer
	if !stack.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&stack, v1alpha1.InferenceStackFinalizer) {
			log.Info("finalizing InferenceStack")
			// Children are garbage collected via OwnerReference.
			// Perform any additional cleanup here.
			controllerutil.RemoveFinalizer(&stack, v1alpha1.InferenceStackFinalizer)
			if err := r.Update(ctx, &stack); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 3. Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&stack, v1alpha1.InferenceStackFinalizer) {
		controllerutil.AddFinalizer(&stack, v1alpha1.InferenceStackFinalizer)
		if err := r.Update(ctx, &stack); err != nil {
			return ctrl.Result{}, err
		}
		// Re-fetch after update
		if err := r.Get(ctx, req.NamespacedName, &stack); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 4. Check spec hash for changes
	currentHash, hashErr := hashSpec(stack.Spec)
	if hashErr != nil {
		log.Error("failed to hash spec", "error", hashErr)
	}
	specChanged := hashErr != nil || currentHash != stack.Status.ObservedSpecHash

	if specChanged {
		log.Info("spec changed, reconciling children", "hash", currentHash)
	}

	// 5-7. Reconcile all children
	var children []v1alpha1.ChildStatus

	children = append(children, r.reconcileInferencePool(ctx, &stack))
	children = append(children, r.reconcileEPPConfig(ctx, &stack))
	children = append(children, r.reconcileAutoscaler(ctx, &stack))
	children = append(children, r.reconcileHTTPRoute(ctx, &stack))
	children = append(children, r.reconcileDCGMExporter(ctx, &stack))

	// 8. Compute aggregate phase
	phase := computePhase(children)

	// 9. Update parent status
	now := metav1.Now()
	stack.Status.Phase = phase
	stack.Status.Children = children
	stack.Status.ObservedSpecHash = currentHash
	stack.Status.LastReconciledAt = &now

	if phase == v1alpha1.PhaseReady {
		v1alpha1.SetCondition(&stack.Status.Conditions, v1alpha1.ConditionReady, metav1.ConditionTrue, "AllChildrenReady", "All child resources are ready")
		v1alpha1.SetCondition(&stack.Status.Conditions, v1alpha1.ConditionReconciled, metav1.ConditionTrue, "ReconcileSucceeded", "Reconciliation completed successfully")
	} else {
		v1alpha1.SetCondition(&stack.Status.Conditions, v1alpha1.ConditionReady, metav1.ConditionFalse, "ChildrenNotReady", "One or more child resources are not ready")
		v1alpha1.SetCondition(&stack.Status.Conditions, v1alpha1.ConditionReconciled, metav1.ConditionTrue, "ReconcileSucceeded", "Reconciliation completed with degraded children")
	}

	if err := r.Status().Update(ctx, &stack); err != nil {
		log.Error("unable to update InferenceStack status", "error", err)
		return ctrl.Result{}, err
	}

	log.Info("reconciliation complete", "phase", phase, "children", len(children))

	// 10. Requeue for drift detection
	return ctrl.Result{RequeueAfter: reconcileInterval}, nil
}

// crdExists checks whether the given GVK is known to the API server.
func crdExists(mgr ctrl.Manager, gvk schema.GroupVersionKind) bool {
	_, err := mgr.GetRESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	return err == nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InferenceStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ownerHandler := handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &v1alpha1.InferenceStack{})

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.InferenceStack{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.DaemonSet{})

	// Conditionally watch InferencePool if the CRD is installed.
	inferencePoolGVK := schema.GroupVersionKind{Group: "inference.networking.x-k8s.io", Version: "v1alpha2", Kind: "InferencePool"}
	if crdExists(mgr, inferencePoolGVK) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(inferencePoolGVK)
		builder = builder.Watches(obj, ownerHandler)
		slog.Info("watching InferencePool CRD")
	} else {
		slog.Warn("InferencePool CRD not found, skipping watch")
	}

	// Conditionally watch KEDA ScaledObject if the CRD is installed.
	scaledObjectGVK := schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"}
	if crdExists(mgr, scaledObjectGVK) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(scaledObjectGVK)
		builder = builder.Watches(obj, ownerHandler)
		slog.Info("watching KEDA ScaledObject CRD")
	} else {
		slog.Warn("KEDA ScaledObject CRD not found, skipping watch")
	}

	// Conditionally watch HTTPRoute if the CRD is installed.
	httpRouteGVK := schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"}
	if crdExists(mgr, httpRouteGVK) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(httpRouteGVK)
		builder = builder.Watches(obj, ownerHandler)
		slog.Info("watching HTTPRoute CRD")
	} else {
		slog.Warn("HTTPRoute CRD not found, skipping watch")
	}

	return builder.Complete(r)
}
