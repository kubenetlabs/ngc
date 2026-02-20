package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/database"
)

// policyGVR maps policy type strings to their GVR in the cluster.
var policyGVR = map[string]schema.GroupVersionResource{
	"ratelimit":      {Group: "gateway.nginx.org", Version: "v1alpha1", Resource: "ratelimitpolicies"},
	"clientsettings": {Group: "gateway.nginx.org", Version: "v1alpha1", Resource: "clientsettingspolicies"},
	"backendtls":     {Group: "gateway.networking.k8s.io", Version: "v1alpha3", Resource: "backendtlspolicies"},
	"observability":  {Group: "gateway.nginx.org", Version: "v1alpha1", Resource: "observabilitypolicies"},
}

// PolicyHandler handles policy API requests (rate-limit, auth, retry, etc.).
type PolicyHandler struct {
	Store database.Store
}

func (h *PolicyHandler) resolvePolicyType(r *http.Request) (schema.GroupVersionResource, string, bool) {
	policyType := chi.URLParam(r, "type")
	gvr, ok := policyGVR[policyType]
	return gvr, policyType, ok
}

// List returns all policies of the given type.
func (h *PolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	gvr, _, ok := h.resolvePolicyType(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown policy type")
		return
	}

	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}
	dc := k8s.DynamicClient()
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	namespace := r.URL.Query().Get("namespace")
	var list *unstructured.UnstructuredList
	var err error
	if namespace != "" {
		list, err = dc.Resource(gvr).Namespace(namespace).List(r.Context(), metav1.ListOptions{})
	} else {
		list, err = dc.Resource(gvr).Namespace("").List(r.Context(), metav1.ListOptions{})
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing policies: %v", err))
		return
	}

	policies := make([]PolicyResponse, 0, len(list.Items))
	for i := range list.Items {
		policies = append(policies, toPolicyResponse(&list.Items[i]))
	}
	writeJSON(w, http.StatusOK, policies)
}

// Get returns a single policy by name.
func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	gvr, _, ok := h.resolvePolicyType(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown policy type")
		return
	}

	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}
	dc := k8s.DynamicClient()
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	obj, err := dc.Resource(gvr).Namespace(namespace).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("policy not found: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, toPolicyResponse(obj))
}

// Create creates a new policy.
func (h *PolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	gvr, policyType, ok := h.resolvePolicyType(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown policy type")
		return
	}

	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}
	dc := k8s.DynamicClient()
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Namespace == "" {
		writeError(w, http.StatusBadRequest, "name and namespace are required")
		return
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvr.Group + "/" + gvr.Version,
			"kind":       policyKind(policyType),
			"metadata": map[string]interface{}{
				"name":      req.Name,
				"namespace": req.Namespace,
			},
			"spec": req.Spec,
		},
	}

	created, err := dc.Resource(gvr).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating policy: %v", err))
		return
	}
	resp := toPolicyResponse(created)
	auditLog(h.Store, r.Context(), "create", "Policy", req.Name, req.Namespace, nil, resp)
	writeJSON(w, http.StatusCreated, resp)
}

// Update modifies an existing policy.
func (h *PolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	gvr, _, ok := h.resolvePolicyType(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown policy type")
		return
	}

	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}
	dc := k8s.DynamicClient()
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	existing, err := dc.Resource(gvr).Namespace(namespace).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("policy not found: %v", err))
		return
	}

	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	beforeResp := toPolicyResponse(existing)
	existing.Object["spec"] = req.Spec

	updated, err := dc.Resource(gvr).Namespace(namespace).Update(r.Context(), existing, metav1.UpdateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating policy: %v", err))
		return
	}
	afterResp := toPolicyResponse(updated)
	auditLog(h.Store, r.Context(), "update", "Policy", name, namespace, beforeResp, afterResp)
	writeJSON(w, http.StatusOK, afterResp)
}

// Delete removes a policy.
func (h *PolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	gvr, _, ok := h.resolvePolicyType(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown policy type")
		return
	}

	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}
	dc := k8s.DynamicClient()
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	if err := dc.Resource(gvr).Namespace(namespace).Delete(r.Context(), name, metav1.DeleteOptions{}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting policy: %v", err))
		return
	}
	auditLog(h.Store, r.Context(), "delete", "Policy", name, namespace, map[string]string{"name": name, "namespace": namespace}, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Conflicts returns detected policy conflicts.
func (h *PolicyHandler) Conflicts(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// PolicyResponse is the API response for a policy resource.
type PolicyResponse struct {
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	PolicyType string                 `json:"policyType"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	CreatedAt  string                 `json:"createdAt"`
}

// CreatePolicyRequest is the request body for creating a policy.
type CreatePolicyRequest struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Spec      map[string]interface{} `json:"spec"`
}

// UpdatePolicyRequest is the request body for updating a policy.
type UpdatePolicyRequest struct {
	Spec map[string]interface{} `json:"spec"`
}

func toPolicyResponse(obj *unstructured.Unstructured) PolicyResponse {
	resp := PolicyResponse{
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		PolicyType: obj.GetKind(),
	}
	if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
		resp.Spec = spec
	}
	if ct := obj.GetCreationTimestamp(); !ct.IsZero() {
		resp.CreatedAt = ct.UTC().Format("2006-01-02T15:04:05Z")
	}
	return resp
}

func policyKind(policyType string) string {
	switch policyType {
	case "ratelimit":
		return "RateLimitPolicy"
	case "clientsettings":
		return "ClientSettingsPolicy"
	case "backendtls":
		return "BackendTLSPolicy"
	case "observability":
		return "ObservabilityPolicy"
	default:
		return policyType
	}
}
