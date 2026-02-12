package controller

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/operator/api/v1alpha1"
)

// reconcileGateway creates or updates the Gateway child resource.
func (r *GatewayBundleReconciler) reconcileGateway(ctx context.Context, bundle *v1alpha1.GatewayBundle) v1alpha1.ChildStatus {
	name := bundle.Name
	log := slog.With("child", "Gateway", "name", name)

	desired := buildDesiredGateway(bundle)

	existing := &gatewayv1.Gateway{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: bundle.Namespace}, existing)

	if errors.IsNotFound(err) {
		log.Info("creating Gateway")
		if err := r.Create(ctx, desired); err != nil {
			log.Error("failed to create Gateway", "error", err)
			return v1alpha1.ChildStatus{Kind: "Gateway", Name: name, Ready: false, Message: fmt.Sprintf("create failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "Gateway", Name: name, Ready: true, Message: "created"}
	}
	if err != nil {
		log.Error("failed to get Gateway", "error", err)
		return v1alpha1.ChildStatus{Kind: "Gateway", Name: name, Ready: false, Message: fmt.Sprintf("get failed: %v", err)}
	}

	// Update spec if drifted
	if specDrifted(existing.Spec, desired.Spec) {
		log.Info("Gateway spec drifted, updating")
		existing.Spec = desired.Spec
		existing.Labels = desired.Labels
		existing.Annotations = desired.Annotations
		if err := r.Update(ctx, existing); err != nil {
			log.Error("failed to update Gateway", "error", err)
			return v1alpha1.ChildStatus{Kind: "Gateway", Name: name, Ready: false, Message: fmt.Sprintf("update failed: %v", err)}
		}
		return v1alpha1.ChildStatus{Kind: "Gateway", Name: name, Ready: true, Message: "updated"}
	}

	// Check Gateway status conditions for readiness
	ready := true
	for _, cond := range existing.Status.Conditions {
		if cond.Type == "Accepted" && string(cond.Status) != "True" {
			ready = false
		}
		if cond.Type == "Programmed" && string(cond.Status) != "True" {
			ready = false
		}
	}

	msg := "in sync"
	if !ready {
		msg = "waiting for gateway controller"
	}

	return v1alpha1.ChildStatus{Kind: "Gateway", Name: name, Ready: ready, Message: msg}
}

// buildDesiredGateway constructs the desired Gateway resource from the GatewayBundle spec.
func buildDesiredGateway(bundle *v1alpha1.GatewayBundle) *gatewayv1.Gateway {
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bundle.Name,
			Namespace: bundle.Namespace,
			Labels: mergeLabels(bundle.Spec.Labels, map[string]string{
				"app.kubernetes.io/managed-by": "ngf-console",
				"ngf-console.f5.com/bundle":    bundle.Name,
			}),
			Annotations: bundle.Spec.Annotations,
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(bundle.Spec.GatewayClassName),
			Listeners:        make([]gatewayv1.Listener, 0, len(bundle.Spec.Listeners)),
		},
	}

	// Set owner reference
	isController := true
	blockDeletion := true
	gw.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         v1alpha1.SchemeGroupVersion.String(),
			Kind:               "GatewayBundle",
			Name:               bundle.Name,
			UID:                bundle.UID,
			Controller:         &isController,
			BlockOwnerDeletion: &blockDeletion,
		},
	})

	// Convert listeners
	for _, l := range bundle.Spec.Listeners {
		listener := gatewayv1.Listener{
			Name:     gatewayv1.SectionName(l.Name),
			Port:     gatewayv1.PortNumber(l.Port),
			Protocol: gatewayv1.ProtocolType(l.Protocol),
		}

		if l.Hostname != "" {
			h := gatewayv1.Hostname(l.Hostname)
			listener.Hostname = &h
		}

		if l.TLS != nil {
			tlsCfg := &gatewayv1.ListenerTLSConfig{}
			if l.TLS.Mode != "" {
				mode := gatewayv1.TLSModeType(l.TLS.Mode)
				tlsCfg.Mode = &mode
			}
			for _, ref := range l.TLS.CertificateRefs {
				certRef := gatewayv1.SecretObjectReference{
					Name: gatewayv1.ObjectName(ref.Name),
				}
				if ref.Namespace != "" {
					ns := gatewayv1.Namespace(ref.Namespace)
					certRef.Namespace = &ns
				}
				tlsCfg.CertificateRefs = append(tlsCfg.CertificateRefs, certRef)
			}
			listener.TLS = tlsCfg
		}

		if l.AllowedRoutes != nil && l.AllowedRoutes.Namespaces != nil {
			ar := &gatewayv1.AllowedRoutes{
				Namespaces: &gatewayv1.RouteNamespaces{},
			}
			if l.AllowedRoutes.Namespaces.From != "" {
				from := gatewayv1.FromNamespaces(l.AllowedRoutes.Namespaces.From)
				ar.Namespaces.From = &from
			}
			listener.AllowedRoutes = ar
		}

		gw.Spec.Listeners = append(gw.Spec.Listeners, listener)
	}

	return gw
}

// mergeLabels merges two label maps, with overrides taking precedence.
func mergeLabels(base, overrides map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overrides {
		result[k] = v
	}
	return result
}
