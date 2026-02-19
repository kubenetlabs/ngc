package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// GatewayHandler handles Gateway and GatewayClass API requests.
// Create, Update, and Delete now operate on GatewayBundle CRDs,
// while List and Get still read Gateway resources directly for live status.
type GatewayHandler struct{}

// getDynamicClient returns the dynamic client from the cluster context.
func (h *GatewayHandler) getDynamicClient(r *http.Request) dynamic.Interface {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		return nil
	}
	return k8s.DynamicClient()
}

// List returns all gateways, optionally filtered by ?namespace= query param.
func (h *GatewayHandler) List(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := r.URL.Query().Get("namespace")
	gateways, err := k8s.ListGateways(r.Context(), ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]GatewayResponse, 0, len(gateways))
	for i := range gateways {
		resp = append(resp, toGatewayResponse(&gateways[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single gateway by namespace and name.
func (h *GatewayHandler) Get(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	gw, err := k8s.GetGateway(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toGatewayResponse(gw))
}

// ListClasses returns all GatewayClasses.
func (h *GatewayHandler) ListClasses(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	classes, err := k8s.ListGatewayClasses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]GatewayClassResponse, 0, len(classes))
	for i := range classes {
		resp = append(resp, toGatewayClassResponse(&classes[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetClass returns a single GatewayClass by name.
func (h *GatewayHandler) GetClass(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")

	gc, err := k8s.GetGatewayClass(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toGatewayClassResponse(gc))
}

// Create creates a new gateway via a GatewayBundle CRD.
func (h *GatewayHandler) Create(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreateGatewayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.Namespace == "" || req.GatewayClassName == "" || len(req.Listeners) == 0 {
		writeError(w, http.StatusBadRequest, "name, namespace, gatewayClassName, and at least one listener are required")
		return
	}

	// Convert the gateway request to a GatewayBundle create request.
	bundleReq := gatewayReqToBundle(req)
	obj := toGatewayBundleUnstructured(bundleReq)

	created, err := dc.Resource(gatewayBundleGVR).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating gatewaybundle: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, toGatewayBundleResponse(created))
}

// Update modifies an existing gateway via the GatewayBundle CRD.
func (h *GatewayHandler) Update(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	// Fetch existing GatewayBundle to get resourceVersion.
	existing, err := dc.Resource(gatewayBundleGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting gatewaybundle %s/%s: %v", ns, name, err))
		return
	}

	var req UpdateGatewayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Convert to GatewayBundle create request for the unstructured builder.
	bundleReq := gatewayUpdateReqToBundle(name, ns, req)
	updated := toGatewayBundleUnstructured(bundleReq)
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

// Delete removes a gateway via the GatewayBundle CRD.
func (h *GatewayHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, map[string]string{"message": "gateway deleted", "name": name, "namespace": ns})
}

// Deploy triggers a gateway deployment by annotating the gateway
// to trigger operator reconciliation.
func (h *GatewayHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	gw, err := k8s.GetGateway(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("gateway not found: %v", err))
		return
	}

	// Add/update annotation to trigger operator reconciliation
	if gw.Annotations == nil {
		gw.Annotations = make(map[string]string)
	}
	gw.Annotations["ngf-console/deploy-requested"] = time.Now().UTC().Format(time.RFC3339)

	if _, err := k8s.UpdateGateway(r.Context(), gw); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating gateway annotation: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":   "deploy triggered",
		"name":      name,
		"namespace": ns,
	})
}

// gatewayReqToBundle converts a CreateGatewayRequest to a CreateGatewayBundleRequest.
func gatewayReqToBundle(req CreateGatewayRequest) CreateGatewayBundleRequest {
	listeners := make([]GatewayBundleListenerReq, 0, len(req.Listeners))
	for _, l := range req.Listeners {
		bl := GatewayBundleListenerReq{
			Name:     l.Name,
			Port:     l.Port,
			Protocol: l.Protocol,
		}
		if l.Hostname != nil {
			bl.Hostname = *l.Hostname
		}
		listeners = append(listeners, bl)
	}
	return CreateGatewayBundleRequest{
		Name:             req.Name,
		Namespace:        req.Namespace,
		GatewayClassName: req.GatewayClassName,
		Listeners:        listeners,
		Labels:           req.Labels,
		Annotations:      req.Annotations,
	}
}

// gatewayUpdateReqToBundle converts an UpdateGatewayRequest to a CreateGatewayBundleRequest
// suitable for the unstructured builder.
func gatewayUpdateReqToBundle(name, namespace string, req UpdateGatewayRequest) CreateGatewayBundleRequest {
	listeners := make([]GatewayBundleListenerReq, 0, len(req.Listeners))
	for _, l := range req.Listeners {
		bl := GatewayBundleListenerReq{
			Name:     l.Name,
			Port:     l.Port,
			Protocol: l.Protocol,
		}
		if l.Hostname != nil {
			bl.Hostname = *l.Hostname
		}
		listeners = append(listeners, bl)
	}
	return CreateGatewayBundleRequest{
		Name:             name,
		Namespace:        namespace,
		GatewayClassName: req.GatewayClassName,
		Listeners:        listeners,
		Labels:           req.Labels,
		Annotations:      req.Annotations,
	}
}
