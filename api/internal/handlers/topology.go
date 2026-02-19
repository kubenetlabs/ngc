package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

// TopologyHandler handles topology graph API requests.
type TopologyHandler struct{}

// buildGraph constructs the full topology graph of gateways, routes, and services.
func (h *TopologyHandler) buildGraph(ctx context.Context, k8s *kubernetes.Client) ([]TopologyNode, []TopologyEdge, error) {
	// List all Gateways across all namespaces.
	gateways, err := k8s.ListGateways(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("listing gateways: %w", err)
	}

	// List all HTTPRoutes across all namespaces.
	routes, err := k8s.ListHTTPRoutes(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("listing httproutes: %w", err)
	}

	// List all Services across all namespaces.
	services, err := k8s.ListServices(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("listing services: %w", err)
	}

	var nodes []TopologyNode
	var edges []TopologyEdge
	edgeCounter := 0

	// Build a set of known service identifiers for edge validation.
	serviceSet := make(map[string]bool)
	for _, svc := range services {
		id := fmt.Sprintf("service/%s/%s", svc.Namespace, svc.Name)
		serviceSet[id] = true
		nodes = append(nodes, TopologyNode{
			ID:        id,
			Type:      "service",
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Status:    "healthy",
			Metadata: map[string]string{
				"clusterIP": svc.Spec.ClusterIP,
				"type":      string(svc.Spec.Type),
			},
		})
	}

	// Build gateway nodes.
	for _, gw := range gateways {
		id := fmt.Sprintf("gateway/%s/%s", gw.Namespace, gw.Name)

		status := "healthy"
		for _, cond := range gw.Status.Conditions {
			if cond.Type == "Accepted" && string(cond.Status) != "True" {
				status = "degraded"
			}
			if cond.Type == "Programmed" && string(cond.Status) != "True" {
				status = "error"
			}
		}

		metadata := map[string]string{
			"gatewayClassName": string(gw.Spec.GatewayClassName),
		}
		if len(gw.Status.Addresses) > 0 {
			metadata["address"] = gw.Status.Addresses[0].Value
		}

		nodes = append(nodes, TopologyNode{
			ID:        id,
			Type:      "gateway",
			Name:      gw.Name,
			Namespace: gw.Namespace,
			Status:    status,
			Metadata:  metadata,
		})
	}

	// Build HTTPRoute nodes and edges.
	for _, hr := range routes {
		routeID := fmt.Sprintf("httproute/%s/%s", hr.Namespace, hr.Name)

		status := "healthy"
		for _, ps := range hr.Status.Parents {
			for _, cond := range ps.Conditions {
				if cond.Type == "Accepted" && string(cond.Status) != "True" {
					status = "degraded"
				}
				if cond.Type == "ResolvedRefs" && string(cond.Status) != "True" {
					status = "error"
				}
			}
		}

		metadata := map[string]string{}
		if len(hr.Spec.Hostnames) > 0 {
			metadata["hostname"] = string(hr.Spec.Hostnames[0])
		}

		nodes = append(nodes, TopologyNode{
			ID:        routeID,
			Type:      "httproute",
			Name:      hr.Name,
			Namespace: hr.Namespace,
			Status:    status,
			Metadata:  metadata,
		})

		// Edges from HTTPRoute to parent Gateways.
		for _, pr := range hr.Spec.ParentRefs {
			gwNamespace := hr.Namespace
			if pr.Namespace != nil {
				gwNamespace = string(*pr.Namespace)
			}
			gwID := fmt.Sprintf("gateway/%s/%s", gwNamespace, string(pr.Name))

			edgeCounter++
			edges = append(edges, TopologyEdge{
				ID:     fmt.Sprintf("edge-%d", edgeCounter),
				Source: routeID,
				Target: gwID,
				Type:   "parentRef",
			})
		}

		// Edges from HTTPRoute to backend Services.
		for _, rule := range hr.Spec.Rules {
			for _, br := range rule.BackendRefs {
				svcNamespace := hr.Namespace
				if br.Namespace != nil {
					svcNamespace = string(*br.Namespace)
				}
				svcID := fmt.Sprintf("service/%s/%s", svcNamespace, string(br.Name))

				// Only add edge if the target service exists in the cluster,
				// or add it anyway to show the reference (the service node
				// might not exist yet if it is misconfigured).
				if !serviceSet[svcID] {
					// Add a placeholder service node for dangling references.
					serviceSet[svcID] = true
					nodes = append(nodes, TopologyNode{
						ID:        svcID,
						Type:      "service",
						Name:      string(br.Name),
						Namespace: svcNamespace,
						Status:    "error",
						Metadata:  map[string]string{"reason": "service not found"},
					})
				}

				edgeCounter++
				edges = append(edges, TopologyEdge{
					ID:     fmt.Sprintf("edge-%d", edgeCounter),
					Source: routeID,
					Target: svcID,
					Type:   "backendRef",
				})
			}
		}
	}

	return nodes, edges, nil
}

// Full returns the full cluster topology graph.
// It lists all Gateways, HTTPRoutes, and Services, then builds a node/edge graph
// showing how routes connect gateways to backend services.
func (h *TopologyHandler) Full(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	nodes, edges, err := h.buildGraph(r.Context(), k8s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ensure non-nil slices in response.
	if nodes == nil {
		nodes = []TopologyNode{}
	}
	if edges == nil {
		edges = []TopologyEdge{}
	}

	writeJSON(w, http.StatusOK, TopologyResponse{Nodes: nodes, Edges: edges})
}

// ByGateway returns the topology graph scoped to a specific gateway.
func (h *TopologyHandler) ByGateway(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")

	nodes, edges, err := h.buildGraph(r.Context(), k8s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the gateway node.
	var gatewayID string
	if namespace != "" {
		gatewayID = fmt.Sprintf("gateway/%s/%s", namespace, name)
	} else {
		// Search all nodes for a gateway matching the name.
		for _, n := range nodes {
			if n.Type == "gateway" && n.Name == name {
				gatewayID = n.ID
				break
			}
		}
	}

	if gatewayID == "" {
		writeError(w, http.StatusNotFound, fmt.Sprintf("gateway %q not found in topology", name))
		return
	}

	// Verify the gateway node exists.
	found := false
	for _, n := range nodes {
		if n.ID == gatewayID {
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("gateway %q not found in topology", name))
		return
	}

	// Build connected set: start with gateway, find routes, then services.
	connectedNodes := map[string]bool{gatewayID: true}

	// Find routes connected to this gateway (edges where Target == gatewayID and Type == "parentRef").
	for _, e := range edges {
		if e.Target == gatewayID && e.Type == "parentRef" {
			connectedNodes[e.Source] = true
		}
	}

	// Find services connected to those routes (edges where Source is a connected route and Type == "backendRef").
	for _, e := range edges {
		if connectedNodes[e.Source] && e.Type == "backendRef" {
			connectedNodes[e.Target] = true
		}
	}

	// Filter nodes and edges.
	var filteredNodes []TopologyNode
	for _, n := range nodes {
		if connectedNodes[n.ID] {
			filteredNodes = append(filteredNodes, n)
		}
	}

	var filteredEdges []TopologyEdge
	for _, e := range edges {
		if connectedNodes[e.Source] && connectedNodes[e.Target] {
			filteredEdges = append(filteredEdges, e)
		}
	}

	if filteredNodes == nil {
		filteredNodes = []TopologyNode{}
	}
	if filteredEdges == nil {
		filteredEdges = []TopologyEdge{}
	}

	writeJSON(w, http.StatusOK, TopologyResponse{Nodes: filteredNodes, Edges: filteredEdges})
}
