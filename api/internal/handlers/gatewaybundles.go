package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

var gatewayBundleGVR = schema.GroupVersionResource{
	Group:    "ngf-console.f5.com",
	Version:  "v1alpha1",
	Resource: "gatewaybundles",
}

// GatewayBundleHandler handles GatewayBundle CRD API requests using the dynamic client.
type GatewayBundleHandler struct {
	DynamicClient dynamic.Interface
}

// getDynamicClient returns the dynamic client from the handler field or falls back
// to the cluster context's dynamic client.
func (h *GatewayBundleHandler) getDynamicClient(r *http.Request) dynamic.Interface {
	if h.DynamicClient != nil {
		return h.DynamicClient
	}
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		return nil
	}
	return k8s.DynamicClient()
}

// List returns all GatewayBundles across all namespaces.
func (h *GatewayBundleHandler) List(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	list, err := dc.Resource(gatewayBundleGVR).Namespace("").List(r.Context(), metav1.ListOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing gatewaybundles: %v", err))
		return
	}

	resp := make([]GatewayBundleResponse, 0, len(list.Items))
	for i := range list.Items {
		resp = append(resp, toGatewayBundleResponse(&list.Items[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single GatewayBundle by namespace and name.
func (h *GatewayBundleHandler) Get(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	obj, err := dc.Resource(gatewayBundleGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting gatewaybundle %s/%s: %v", ns, name, err))
		return
	}
	writeJSON(w, http.StatusOK, toGatewayBundleResponse(obj))
}

// Create creates a new GatewayBundle from the JSON request body.
func (h *GatewayBundleHandler) Create(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreateGatewayBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.Namespace == "" || req.GatewayClassName == "" || len(req.Listeners) == 0 {
		writeError(w, http.StatusBadRequest, "name, namespace, gatewayClassName, and at least one listener are required")
		return
	}

	obj := toGatewayBundleUnstructured(req)
	created, err := dc.Resource(gatewayBundleGVR).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating gatewaybundle: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, toGatewayBundleResponse(created))
}

// Update modifies an existing GatewayBundle by namespace and name.
func (h *GatewayBundleHandler) Update(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	// Fetch existing to get resourceVersion.
	existing, err := dc.Resource(gatewayBundleGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting gatewaybundle %s/%s: %v", ns, name, err))
		return
	}

	var req UpdateGatewayBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Build an unstructured object from the update request, preserving identity from existing.
	createReq := CreateGatewayBundleRequest{
		Name:             name,
		Namespace:        ns,
		GatewayClassName: req.GatewayClassName,
		Listeners:        req.Listeners,
		Labels:           req.Labels,
		Annotations:      req.Annotations,
		NginxProxy:       req.NginxProxy,
		WAF:              req.WAF,
		SnippetsFilter:   req.SnippetsFilter,
		TLS:              req.TLS,
	}

	updated := toGatewayBundleUnstructured(createReq)
	updated.SetNamespace(ns)
	updated.SetName(name)
	updated.SetResourceVersion(existing.GetResourceVersion())
	updated.SetUID(existing.GetUID())
	updated.SetCreationTimestamp(existing.GetCreationTimestamp())

	result, err := dc.Resource(gatewayBundleGVR).Namespace(ns).Update(r.Context(), updated, metav1.UpdateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating gatewaybundle %s/%s: %v", ns, name, err))
		return
	}
	writeJSON(w, http.StatusOK, toGatewayBundleResponse(result))
}

// Delete removes a GatewayBundle by namespace and name.
func (h *GatewayBundleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	err := dc.Resource(gatewayBundleGVR).Namespace(ns).Delete(r.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting gatewaybundle %s/%s: %v", ns, name, err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "gatewaybundle deleted", "name": name, "namespace": ns})
}

// GetStatus returns just the status sub-resource of a GatewayBundle.
func (h *GatewayBundleHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	obj, err := dc.Resource(gatewayBundleGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{}, "status")
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting gatewaybundle status %s/%s: %v", ns, name, err))
		return
	}

	resp := toGatewayBundleResponse(obj)
	statusResp := map[string]any{
		"name":           resp.Name,
		"namespace":      resp.Namespace,
		"phase":          resp.Phase,
		"children":       resp.Children,
		"conditions":     resp.Conditions,
		"gatewayAddress": resp.GatewayAddress,
	}
	if resp.ObservedSpecHash != "" {
		statusResp["observedSpecHash"] = resp.ObservedSpecHash
	}
	if resp.LastReconciledAt != "" {
		statusResp["lastReconciledAt"] = resp.LastReconciledAt
	}
	writeJSON(w, http.StatusOK, statusResp)
}

// toGatewayBundleResponse converts an unstructured GatewayBundle to a response type.
func toGatewayBundleResponse(obj *unstructured.Unstructured) GatewayBundleResponse {
	resp := GatewayBundleResponse{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Labels:    obj.GetLabels(),
		CreatedAt: obj.GetCreationTimestamp().UTC().Format("2006-01-02T15:04:05Z"),
	}

	// Annotations (filter out kubectl.kubernetes.io managed fields)
	annotations := obj.GetAnnotations()
	if len(annotations) > 0 {
		resp.Annotations = annotations
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec != nil {
		resp.GatewayClassName, _, _ = unstructured.NestedString(spec, "gatewayClassName")

		// Listeners
		listeners, _, _ := unstructured.NestedSlice(spec, "listeners")
		for _, l := range listeners {
			lMap, ok := l.(map[string]any)
			if !ok {
				continue
			}
			lr := GatewayBundleListenerResp{}
			lr.Name, _, _ = unstructured.NestedString(lMap, "name")
			port, _, _ := unstructured.NestedInt64(lMap, "port")
			lr.Port = int32(port)
			lr.Protocol, _, _ = unstructured.NestedString(lMap, "protocol")
			lr.Hostname, _, _ = unstructured.NestedString(lMap, "hostname")

			// Listener TLS
			tlsMap, tlsFound, _ := unstructured.NestedMap(lMap, "tls")
			if tlsFound && tlsMap != nil {
				tls := &ListenerTLSResp{}
				tls.Mode, _, _ = unstructured.NestedString(tlsMap, "mode")
				certRefs, _, _ := unstructured.NestedSlice(tlsMap, "certificateRefs")
				for _, cr := range certRefs {
					crMap, ok := cr.(map[string]any)
					if !ok {
						continue
					}
					ref := CertRefResp{}
					ref.Name, _, _ = unstructured.NestedString(crMap, "name")
					ref.Namespace, _, _ = unstructured.NestedString(crMap, "namespace")
					tls.CertificateRefs = append(tls.CertificateRefs, ref)
				}
				lr.TLS = tls
			}

			// AllowedRoutes
			arMap, arFound, _ := unstructured.NestedMap(lMap, "allowedRoutes")
			if arFound && arMap != nil {
				ar := &AllowedRoutesResp{}
				nsMap, nsFound, _ := unstructured.NestedMap(arMap, "namespaces")
				if nsFound && nsMap != nil {
					ns := &NamespaceResp{}
					ns.From, _, _ = unstructured.NestedString(nsMap, "from")
					selector, _, _ := unstructured.NestedStringMap(nsMap, "selector")
					if len(selector) > 0 {
						ns.Selector = selector
					}
					ar.Namespaces = ns
				}
				lr.AllowedRoutes = ar
			}

			resp.Listeners = append(resp.Listeners, lr)
		}

		// NginxProxy
		npMap, npFound, _ := unstructured.NestedMap(spec, "nginxProxy")
		if npFound && npMap != nil {
			np := &NginxProxyResp{}
			np.Enabled, _, _ = unstructured.NestedBool(npMap, "enabled")
			np.IPFamily, _, _ = unstructured.NestedString(npMap, "ipFamily")

			rcipMap, rcipFound, _ := unstructured.NestedMap(npMap, "rewriteClientIP")
			if rcipFound && rcipMap != nil {
				rcip := &RewriteClientIPResp{}
				rcip.Mode, _, _ = unstructured.NestedString(rcipMap, "mode")
				rcip.SetIPRecursively, _, _ = unstructured.NestedBool(rcipMap, "setIPRecursively")
				np.RewriteClientIP = rcip
			}

			telMap, telFound, _ := unstructured.NestedMap(npMap, "telemetry")
			if telFound && telMap != nil {
				tel := &NginxTelemetryResp{}
				expMap, expFound, _ := unstructured.NestedMap(telMap, "exporter")
				if expFound && expMap != nil {
					exp := &OTelExporterResp{}
					exp.Endpoint, _, _ = unstructured.NestedString(expMap, "endpoint")
					tel.Exporter = exp
				}
				np.Telemetry = tel
			}

			resp.NginxProxy = np
		}

		// WAF
		wafMap, wafFound, _ := unstructured.NestedMap(spec, "waf")
		if wafFound && wafMap != nil {
			waf := &WAFResp{}
			waf.Enabled, _, _ = unstructured.NestedBool(wafMap, "enabled")
			waf.PolicyRef, _, _ = unstructured.NestedString(wafMap, "policyRef")
			resp.WAF = waf
		}

		// SnippetsFilter
		sfMap, sfFound, _ := unstructured.NestedMap(spec, "snippetsFilter")
		if sfFound && sfMap != nil {
			sf := &SnippetsFilterResp{}
			sf.Enabled, _, _ = unstructured.NestedBool(sfMap, "enabled")
			sf.ServerSnippet, _, _ = unstructured.NestedString(sfMap, "serverSnippet")
			sf.LocationSnippet, _, _ = unstructured.NestedString(sfMap, "locationSnippet")
			resp.SnippetsFilter = sf
		}

		// TLS
		tlsMap, tlsFound, _ := unstructured.NestedMap(spec, "tls")
		if tlsFound && tlsMap != nil {
			tls := &GatewayTLSResp{}
			secretRefs, _, _ := unstructured.NestedSlice(tlsMap, "secretRefs")
			for _, sr := range secretRefs {
				srMap, ok := sr.(map[string]any)
				if !ok {
					continue
				}
				ref := TLSSecretRefResp{}
				ref.Name, _, _ = unstructured.NestedString(srMap, "name")
				ref.Namespace, _, _ = unstructured.NestedString(srMap, "namespace")
				tls.SecretRefs = append(tls.SecretRefs, ref)
			}
			resp.TLS = tls
		}
	}

	// Status
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status != nil {
		resp.Phase, _, _ = unstructured.NestedString(status, "phase")
		resp.GatewayAddress, _, _ = unstructured.NestedString(status, "gatewayAddress")
		resp.ObservedSpecHash, _, _ = unstructured.NestedString(status, "observedSpecHash")
		resp.LastReconciledAt, _, _ = unstructured.NestedString(status, "lastReconciledAt")

		// Children
		children, _, _ := unstructured.NestedSlice(status, "children")
		for _, child := range children {
			childMap, ok := child.(map[string]any)
			if !ok {
				continue
			}
			cs := ChildStatusResponse{}
			cs.Kind, _, _ = unstructured.NestedString(childMap, "kind")
			cs.Name, _, _ = unstructured.NestedString(childMap, "name")
			cs.Ready, _, _ = unstructured.NestedBool(childMap, "ready")
			cs.Message, _, _ = unstructured.NestedString(childMap, "message")
			resp.Children = append(resp.Children, cs)
		}

		// Conditions
		conditions, _, _ := unstructured.NestedSlice(status, "conditions")
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]any)
			if !ok {
				continue
			}
			cr := ConditionResponse{}
			cr.Type, _, _ = unstructured.NestedString(condMap, "type")
			cr.Status, _, _ = unstructured.NestedString(condMap, "status")
			cr.Reason, _, _ = unstructured.NestedString(condMap, "reason")
			cr.Message, _, _ = unstructured.NestedString(condMap, "message")
			cr.LastTransitionTime, _, _ = unstructured.NestedString(condMap, "lastTransitionTime")
			resp.Conditions = append(resp.Conditions, cr)
		}
	}

	return resp
}

// toGatewayBundleUnstructured converts a create request into an unstructured object.
func toGatewayBundleUnstructured(req CreateGatewayBundleRequest) *unstructured.Unstructured {
	listeners := make([]any, 0, len(req.Listeners))
	for _, l := range req.Listeners {
		lMap := map[string]any{
			"name":     l.Name,
			"port":     int64(l.Port),
			"protocol": l.Protocol,
		}
		if l.Hostname != "" {
			lMap["hostname"] = l.Hostname
		}
		if l.TLS != nil {
			tlsMap := map[string]any{}
			if l.TLS.Mode != "" {
				tlsMap["mode"] = l.TLS.Mode
			}
			if len(l.TLS.CertificateRefs) > 0 {
				certRefs := make([]any, 0, len(l.TLS.CertificateRefs))
				for _, cr := range l.TLS.CertificateRefs {
					crMap := map[string]any{"name": cr.Name}
					if cr.Namespace != "" {
						crMap["namespace"] = cr.Namespace
					}
					certRefs = append(certRefs, crMap)
				}
				tlsMap["certificateRefs"] = certRefs
			}
			lMap["tls"] = tlsMap
		}
		if l.AllowedRoutes != nil {
			arMap := map[string]any{}
			if l.AllowedRoutes.Namespaces != nil {
				nsMap := map[string]any{}
				if l.AllowedRoutes.Namespaces.From != "" {
					nsMap["from"] = l.AllowedRoutes.Namespaces.From
				}
				if len(l.AllowedRoutes.Namespaces.Selector) > 0 {
					nsMap["selector"] = toStringAnyMap(l.AllowedRoutes.Namespaces.Selector)
				}
				arMap["namespaces"] = nsMap
			}
			lMap["allowedRoutes"] = arMap
		}
		listeners = append(listeners, lMap)
	}

	spec := map[string]any{
		"gatewayClassName": req.GatewayClassName,
		"listeners":        listeners,
	}

	if len(req.Labels) > 0 {
		spec["labels"] = toStringAnyMap(req.Labels)
	}
	if len(req.Annotations) > 0 {
		spec["annotations"] = toStringAnyMap(req.Annotations)
	}

	if req.NginxProxy != nil {
		npMap := map[string]any{
			"enabled": req.NginxProxy.Enabled,
		}
		if req.NginxProxy.IPFamily != "" {
			npMap["ipFamily"] = req.NginxProxy.IPFamily
		}
		if req.NginxProxy.RewriteClientIP != nil {
			rcipMap := map[string]any{}
			if req.NginxProxy.RewriteClientIP.Mode != "" {
				rcipMap["mode"] = req.NginxProxy.RewriteClientIP.Mode
			}
			rcipMap["setIPRecursively"] = req.NginxProxy.RewriteClientIP.SetIPRecursively
			npMap["rewriteClientIP"] = rcipMap
		}
		if req.NginxProxy.Telemetry != nil {
			telMap := map[string]any{}
			if req.NginxProxy.Telemetry.Exporter != nil {
				expMap := map[string]any{}
				if req.NginxProxy.Telemetry.Exporter.Endpoint != "" {
					expMap["endpoint"] = req.NginxProxy.Telemetry.Exporter.Endpoint
				}
				telMap["exporter"] = expMap
			}
			npMap["telemetry"] = telMap
		}
		spec["nginxProxy"] = npMap
	}

	if req.WAF != nil {
		wafMap := map[string]any{
			"enabled": req.WAF.Enabled,
		}
		if req.WAF.PolicyRef != "" {
			wafMap["policyRef"] = req.WAF.PolicyRef
		}
		spec["waf"] = wafMap
	}

	if req.SnippetsFilter != nil {
		sfMap := map[string]any{
			"enabled": req.SnippetsFilter.Enabled,
		}
		if req.SnippetsFilter.ServerSnippet != "" {
			sfMap["serverSnippet"] = req.SnippetsFilter.ServerSnippet
		}
		if req.SnippetsFilter.LocationSnippet != "" {
			sfMap["locationSnippet"] = req.SnippetsFilter.LocationSnippet
		}
		spec["snippetsFilter"] = sfMap
	}

	if req.TLS != nil {
		tlsMap := map[string]any{}
		if len(req.TLS.SecretRefs) > 0 {
			secretRefs := make([]any, 0, len(req.TLS.SecretRefs))
			for _, sr := range req.TLS.SecretRefs {
				srMap := map[string]any{"name": sr.Name}
				if sr.Namespace != "" {
					srMap["namespace"] = sr.Namespace
				}
				secretRefs = append(secretRefs, srMap)
			}
			tlsMap["secretRefs"] = secretRefs
		}
		spec["tls"] = tlsMap
	}

	metadata := map[string]any{
		"name":      req.Name,
		"namespace": req.Namespace,
	}
	if len(req.Labels) > 0 {
		metadata["labels"] = toStringAnyMap(req.Labels)
	}
	if len(req.Annotations) > 0 {
		metadata["annotations"] = toStringAnyMap(req.Annotations)
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": gatewayBundleGVR.Group + "/" + gatewayBundleGVR.Version,
			"kind":       "GatewayBundle",
			"metadata":   metadata,
			"spec":       spec,
		},
	}

	return obj
}

// toStringAnyMap converts map[string]string to map[string]any for unstructured objects.
func toStringAnyMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
