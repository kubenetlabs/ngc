package controller

import (
	"context"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

// GatewayBundleReconciler reconciles GatewayBundle objects.
type GatewayBundleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles reconciliation of GatewayBundle resources.
func (r *GatewayBundleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := slog.With("controller", "GatewayBundle", "name", req.Name, "namespace", req.Namespace)

	// 1. Fetch parent CRD
	var bundle v1alpha1.GatewayBundle
	if err := r.Get(ctx, req.NamespacedName, &bundle); err != nil {
		if errors.IsNotFound(err) {
			log.Info("GatewayBundle not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error("unable to fetch GatewayBundle", "error", err)
		return ctrl.Result{}, err
	}

	// 2. Handle deletion via finalizer
	if !bundle.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&bundle, v1alpha1.GatewayBundleFinalizer) {
			log.Info("finalizing GatewayBundle")
			controllerutil.RemoveFinalizer(&bundle, v1alpha1.GatewayBundleFinalizer)
			if err := r.Update(ctx, &bundle); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 3. Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&bundle, v1alpha1.GatewayBundleFinalizer) {
		controllerutil.AddFinalizer(&bundle, v1alpha1.GatewayBundleFinalizer)
		if err := r.Update(ctx, &bundle); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, &bundle); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 4. Check spec hash for changes
	currentHash, hashErr := hashSpec(bundle.Spec)
	if hashErr != nil {
		log.Error("failed to hash spec", "error", hashErr)
	}
	specChanged := hashErr != nil || currentHash != bundle.Status.ObservedSpecHash

	if specChanged {
		log.Info("spec changed, reconciling children", "hash", currentHash)
	}

	// 5. Reconcile children
	var children []v1alpha1.ChildStatus

	gwStatus := r.reconcileGateway(ctx, &bundle)
	children = append(children, gwStatus)

	npStatus := r.reconcileNginxProxy(ctx, &bundle)
	children = append(children, npStatus)

	wafStatus := r.reconcileWAF(ctx, &bundle)
	children = append(children, wafStatus)

	sfStatus := r.reconcileSnippetsFilter(ctx, &bundle)
	children = append(children, sfStatus)

	tlsStatus := r.reconcileTLSSecrets(ctx, &bundle)
	children = append(children, tlsStatus)

	// 6. Compute aggregate phase
	phase := computePhase(children)

	// 7. Update parent status
	now := metav1.Now()
	bundle.Status.Phase = phase
	bundle.Status.Children = children
	bundle.Status.ObservedSpecHash = currentHash
	bundle.Status.LastReconciledAt = &now

	// Capture gateway address from the Gateway child status
	bundle.Status.GatewayAddress = r.getGatewayAddress(ctx, &bundle)

	if phase == v1alpha1.PhaseReady {
		v1alpha1.SetCondition(&bundle.Status.Conditions, v1alpha1.ConditionReady, metav1.ConditionTrue, "AllChildrenReady", "All child resources are ready")
		v1alpha1.SetCondition(&bundle.Status.Conditions, v1alpha1.ConditionReconciled, metav1.ConditionTrue, "ReconcileSucceeded", "Reconciliation completed successfully")
	} else {
		v1alpha1.SetCondition(&bundle.Status.Conditions, v1alpha1.ConditionReady, metav1.ConditionFalse, "ChildrenNotReady", "One or more child resources are not ready")
		v1alpha1.SetCondition(&bundle.Status.Conditions, v1alpha1.ConditionReconciled, metav1.ConditionTrue, "ReconcileSucceeded", "Reconciliation completed with degraded children")
	}

	if err := r.Status().Update(ctx, &bundle); err != nil {
		log.Error("unable to update GatewayBundle status", "error", err)
		return ctrl.Result{}, err
	}

	log.Info("reconciliation complete", "phase", phase, "children", len(children))

	return ctrl.Result{RequeueAfter: reconcileInterval}, nil
}

// getGatewayAddress reads the address from the Gateway child status.
func (r *GatewayBundleReconciler) getGatewayAddress(ctx context.Context, bundle *v1alpha1.GatewayBundle) string {
	var gw gatewayv1.Gateway
	key := client.ObjectKey{Namespace: bundle.Namespace, Name: bundle.Name}
	if err := r.Get(ctx, key, &gw); err != nil {
		return ""
	}
	if len(gw.Status.Addresses) > 0 {
		return gw.Status.Addresses[0].Value
	}
	return ""
}

// reconcileNginxProxy is a stub for Enterprise NginxProxy reconciliation.
func (r *GatewayBundleReconciler) reconcileNginxProxy(_ context.Context, bundle *v1alpha1.GatewayBundle) v1alpha1.ChildStatus {
	if bundle.Spec.NginxProxy == nil || !bundle.Spec.NginxProxy.Enabled {
		return v1alpha1.ChildStatus{Kind: "NginxProxy", Name: bundle.Name + "-proxy", Ready: true, Message: "not configured"}
	}
	return v1alpha1.ChildStatus{Kind: "NginxProxy", Name: bundle.Name + "-proxy", Ready: true, Message: "not configured (enterprise)"}
}

// reconcileWAF is a stub for Enterprise WAF reconciliation.
func (r *GatewayBundleReconciler) reconcileWAF(_ context.Context, bundle *v1alpha1.GatewayBundle) v1alpha1.ChildStatus {
	if bundle.Spec.WAF == nil || !bundle.Spec.WAF.Enabled {
		return v1alpha1.ChildStatus{Kind: "WAFPolicy", Name: bundle.Name + "-waf", Ready: true, Message: "not configured"}
	}
	return v1alpha1.ChildStatus{Kind: "WAFPolicy", Name: bundle.Name + "-waf", Ready: true, Message: "not configured (enterprise)"}
}

// reconcileSnippetsFilter is a stub for Enterprise SnippetsFilter reconciliation.
func (r *GatewayBundleReconciler) reconcileSnippetsFilter(_ context.Context, bundle *v1alpha1.GatewayBundle) v1alpha1.ChildStatus {
	if bundle.Spec.SnippetsFilter == nil || !bundle.Spec.SnippetsFilter.Enabled {
		return v1alpha1.ChildStatus{Kind: "SnippetsFilter", Name: bundle.Name + "-snippets", Ready: true, Message: "not configured"}
	}
	return v1alpha1.ChildStatus{Kind: "SnippetsFilter", Name: bundle.Name + "-snippets", Ready: true, Message: "not configured (enterprise)"}
}

// reconcileTLSSecrets is a stub for TLS secret management.
func (r *GatewayBundleReconciler) reconcileTLSSecrets(_ context.Context, bundle *v1alpha1.GatewayBundle) v1alpha1.ChildStatus {
	if bundle.Spec.TLS == nil || len(bundle.Spec.TLS.SecretRefs) == 0 {
		return v1alpha1.ChildStatus{Kind: "Secret", Name: bundle.Name + "-tls", Ready: true, Message: "not configured"}
	}
	return v1alpha1.ChildStatus{Kind: "Secret", Name: bundle.Name + "-tls", Ready: true, Message: "not configured"}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayBundleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GatewayBundle{}).
		Owns(&gatewayv1.Gateway{}).
		Complete(r)
}
