package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// ClusterResolver is middleware that extracts the cluster from the URL or
// falls back to the default cluster. It stores the resolved client and
// cluster name in the request context.
func ClusterResolver(mgr *cluster.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clusterName := chi.URLParam(r, "cluster")

			if clusterName != "" {
				k8sClient, err := mgr.Get(clusterName)
				if err != nil {
					writeMiddlewareError(w, http.StatusNotFound, "cluster not found: "+clusterName)
					return
				}
				ctx := cluster.WithClient(r.Context(), k8sClient)
				ctx = cluster.WithClusterName(ctx, clusterName)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				clusterName = mgr.DefaultName()
				k8sClient, err := mgr.Default()
				if err != nil {
					writeMiddlewareError(w, http.StatusServiceUnavailable, "no default cluster available")
					return
				}
				ctx := cluster.WithClient(r.Context(), k8sClient)
				ctx = cluster.WithClusterName(ctx, clusterName)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
		})
	}
}

func writeMiddlewareError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
