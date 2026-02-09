package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/kubenetlabs/ngc/api/internal/handlers"
)

// Server is the main HTTP server for the NGF Console API.
type Server struct {
	Router chi.Router
}

// New creates a new Server with all routes and middleware configured.
func New() *Server {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(RequestLogger)
	r.Use(CORSMiddleware)
	r.Use(chimw.Recoverer)

	s := &Server{Router: r}
	s.registerRoutes()

	return s
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	slog.Info("listening", "addr", addr)
	return http.ListenAndServe(addr, s.Router)
}

// registerRoutes mounts all API v1 route groups.
func (s *Server) registerRoutes() {
	gw := &handlers.GatewayHandler{}
	rt := &handlers.RouteHandler{}
	pol := &handlers.PolicyHandler{}
	cert := &handlers.CertificateHandler{}
	met := &handlers.MetricsHandler{}
	lg := &handlers.LogHandler{}
	topo := &handlers.TopologyHandler{}
	diag := &handlers.DiagnosticsHandler{}
	inf := &handlers.InferenceHandler{}
	infMet := &handlers.InferenceMetricsHandler{}
	infDiag := &handlers.InferenceDiagHandler{}
	coex := &handlers.CoexistenceHandler{}
	xc := &handlers.XCHandler{}
	mig := &handlers.MigrationHandler{}
	aud := &handlers.AuditHandler{}

	s.Router.Route("/api/v1", func(r chi.Router) {
		// Gateway Classes
		r.Route("/gatewayclasses", func(r chi.Router) {
			r.Get("/", gw.List)
			r.Get("/{name}", gw.Get)
		})

		// Gateways
		r.Route("/gateways", func(r chi.Router) {
			r.Get("/", gw.List)
			r.Post("/", gw.Create)
			r.Get("/{name}", gw.Get)
			r.Put("/{name}", gw.Update)
			r.Delete("/{name}", gw.Delete)
			r.Post("/{name}/deploy", gw.Deploy)
		})

		// HTTP Routes
		r.Route("/httproutes", func(r chi.Router) {
			r.Get("/", rt.List)
			r.Post("/", rt.Create)
			r.Get("/{name}", rt.Get)
			r.Put("/{name}", rt.Update)
			r.Delete("/{name}", rt.Delete)
			r.Post("/{name}/simulate", rt.Simulate)
		})

		// gRPC Routes
		r.Route("/grpcroutes", func(r chi.Router) {
			r.Get("/", rt.List)
			r.Post("/", rt.Create)
			r.Get("/{name}", rt.Get)
			r.Put("/{name}", rt.Update)
			r.Delete("/{name}", rt.Delete)
		})

		// TLS Routes
		r.Route("/tlsroutes", func(r chi.Router) {
			r.Get("/", rt.List)
			r.Post("/", rt.Create)
			r.Get("/{name}", rt.Get)
			r.Put("/{name}", rt.Update)
			r.Delete("/{name}", rt.Delete)
		})

		// TCP Routes
		r.Route("/tcproutes", func(r chi.Router) {
			r.Get("/", rt.List)
			r.Post("/", rt.Create)
			r.Get("/{name}", rt.Get)
			r.Put("/{name}", rt.Update)
			r.Delete("/{name}", rt.Delete)
		})

		// UDP Routes
		r.Route("/udproutes", func(r chi.Router) {
			r.Get("/", rt.List)
			r.Post("/", rt.Create)
			r.Get("/{name}", rt.Get)
			r.Put("/{name}", rt.Update)
			r.Delete("/{name}", rt.Delete)
		})

		// Policies
		r.Route("/policies/{type}", func(r chi.Router) {
			r.Get("/", pol.List)
			r.Post("/", pol.Create)
			r.Get("/{name}", pol.Get)
			r.Put("/{name}", pol.Update)
			r.Delete("/{name}", pol.Delete)
			r.Get("/conflicts", pol.Conflicts)
		})

		// Certificates
		r.Route("/certificates", func(r chi.Router) {
			r.Get("/", cert.List)
			r.Post("/", cert.Create)
			r.Get("/expiring", cert.Expiring)
			r.Get("/{name}", cert.Get)
			r.Delete("/{name}", cert.Delete)
		})

		// Metrics
		r.Route("/metrics", func(r chi.Router) {
			r.Get("/summary", met.Summary)
			r.Get("/by-route", met.ByRoute)
			r.Get("/by-gateway", met.ByGateway)
		})

		// Logs
		r.Route("/logs", func(r chi.Router) {
			r.Post("/query", lg.Query)
			r.Get("/topn", lg.TopN)
		})

		// Topology
		r.Route("/topology", func(r chi.Router) {
			r.Get("/full", topo.Full)
			r.Get("/by-gateway/{name}", topo.ByGateway)
		})

		// Diagnostics
		r.Route("/diagnostics", func(r chi.Router) {
			r.Post("/route-check", diag.RouteCheck)
			r.Post("/trace", diag.Trace)
		})

		// Inference
		r.Route("/inference", func(r chi.Router) {
			// Pools
			r.Route("/pools", func(r chi.Router) {
				r.Get("/", inf.ListPools)
				r.Post("/", inf.CreatePool)
				r.Get("/{name}", inf.GetPool)
				r.Put("/{name}", inf.UpdatePool)
				r.Delete("/{name}", inf.DeletePool)
				r.Post("/{name}/deploy", inf.DeployPool)
			})

			// EPP
			r.Get("/epp", inf.GetEPP)
			r.Put("/epp", inf.UpdateEPP)

			// Autoscaling
			r.Get("/autoscaling", inf.GetAutoscaling)
			r.Put("/autoscaling", inf.UpdateAutoscaling)

			// Inference Metrics
			r.Route("/metrics", func(r chi.Router) {
				r.Get("/summary", infMet.Summary)
				r.Get("/by-pool", infMet.ByPool)
				r.Get("/pods", infMet.PodMetrics)
				r.Get("/cost", infMet.Cost)
			})

			// Inference Diagnostics
			r.Route("/diagnostics", func(r chi.Router) {
				r.Get("/slow", infDiag.SlowInference)
				r.Post("/replay", infDiag.Replay)
				r.Post("/benchmark", infDiag.Benchmark)
			})
		})

		// Coexistence
		r.Route("/coexistence", func(r chi.Router) {
			r.Get("/overview", coex.Overview)
			r.Get("/migration-readiness", coex.MigrationReadiness)
		})

		// Cross-Cluster (XC)
		r.Route("/xc", func(r chi.Router) {
			r.Get("/status", xc.Status)
			r.Post("/publish", xc.Publish)
			r.Get("/publish/{id}", xc.GetPublish)
			r.Delete("/publish/{id}", xc.DeletePublish)
			r.Get("/metrics", xc.Metrics)
		})

		// Migration
		r.Route("/migration", func(r chi.Router) {
			r.Post("/import", mig.Import)
			r.Post("/analysis", mig.Analysis)
			r.Post("/generate", mig.Generate)
			r.Post("/apply", mig.Apply)
			r.Post("/validate", mig.Validate)
		})

		// Audit
		r.Route("/audit", func(r chi.Router) {
			r.Get("/", aud.List)
			r.Get("/diff/{id}", aud.Diff)
		})

		// WebSocket
		r.Get("/ws", HandleWebSocket)

		// Events
		r.Get("/events", HandleWebSocket)
	})
}
