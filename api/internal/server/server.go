package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/handlers"
	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// Config holds server dependencies.
type Config struct {
	ClusterManager  *cluster.Manager
	MetricsProvider inference.MetricsProvider
}

// Server is the main HTTP server for the NGF Console API.
type Server struct {
	Router chi.Router
	Config Config
	Hub    *Hub
}

// New creates a new Server with all routes and middleware configured.
func New(cfg Config) *Server {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(RequestLogger)
	r.Use(CORSMiddleware)
	r.Use(chimw.Recoverer)

	hub := NewHub()
	RegisterInferenceTopics(hub)
	hub.Start()

	s := &Server{Router: r, Config: cfg, Hub: hub}
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
	cfgHandler := &handlers.ConfigHandler{}
	clusterHandler := &handlers.ClusterHandler{Manager: s.Config.ClusterManager}
	pol := &handlers.PolicyHandler{}
	cert := &handlers.CertificateHandler{}
	met := &handlers.MetricsHandler{}
	lg := &handlers.LogHandler{}
	topo := &handlers.TopologyHandler{}
	diag := &handlers.DiagnosticsHandler{}
	inf := &handlers.InferenceHandler{Provider: s.Config.MetricsProvider}
	infMet := &handlers.InferenceMetricsHandler{Provider: s.Config.MetricsProvider}
	infDiag := &handlers.InferenceDiagHandler{}
	coex := &handlers.CoexistenceHandler{}
	xc := &handlers.XCHandler{}
	mig := &handlers.MigrationHandler{}
	aud := &handlers.AuditHandler{}

	s.Router.Route("/api/v1", func(r chi.Router) {
		// Cluster management (no cluster middleware needed)
		r.Get("/clusters", clusterHandler.List)

		// Cluster-scoped routes (new multi-cluster paths)
		r.Route("/clusters/{cluster}", func(r chi.Router) {
			r.Use(ClusterResolver(s.Config.ClusterManager))
			s.mountResourceRoutes(r, gw, rt, cfgHandler, pol, cert, met, lg, topo, diag, inf, infMet, infDiag, coex, xc, mig, aud)
		})

		// Legacy routes (backward compat — uses default cluster)
		r.Group(func(r chi.Router) {
			r.Use(ClusterResolver(s.Config.ClusterManager))
			s.mountResourceRoutes(r, gw, rt, cfgHandler, pol, cert, met, lg, topo, diag, inf, infMet, infDiag, coex, xc, mig, aud)
		})

		// WebSocket
		r.Get("/ws", HandleWebSocket)

		// Events
		r.Get("/events", HandleWebSocket)

		// Inference WebSocket topics
		r.Get("/ws/inference/epp-decisions", s.Hub.ServeWS("epp-decisions"))
		r.Get("/ws/inference/gpu-metrics", s.Hub.ServeWS("gpu-metrics"))
		r.Get("/ws/inference/scaling-events", s.Hub.ServeWS("scaling-events"))
	})
}

// mountResourceRoutes registers all resource routes on the given router.
func (s *Server) mountResourceRoutes(
	r chi.Router,
	gw *handlers.GatewayHandler,
	rt *handlers.RouteHandler,
	cfgHandler *handlers.ConfigHandler,
	pol *handlers.PolicyHandler,
	cert *handlers.CertificateHandler,
	met *handlers.MetricsHandler,
	lg *handlers.LogHandler,
	topo *handlers.TopologyHandler,
	diag *handlers.DiagnosticsHandler,
	inf *handlers.InferenceHandler,
	infMet *handlers.InferenceMetricsHandler,
	infDiag *handlers.InferenceDiagHandler,
	coex *handlers.CoexistenceHandler,
	xc *handlers.XCHandler,
	mig *handlers.MigrationHandler,
	aud *handlers.AuditHandler,
) {
	// Config
	r.Get("/config", cfgHandler.GetConfig)

	// Gateway Classes (cluster-scoped, separate handlers)
	r.Route("/gatewayclasses", func(r chi.Router) {
		r.Get("/", gw.ListClasses)
		r.Get("/{name}", gw.GetClass)
	})

	// Gateways (namespace-aware)
	r.Route("/gateways", func(r chi.Router) {
		r.Get("/", gw.List)
		r.Post("/", gw.Create)
		r.Get("/{namespace}/{name}", gw.Get)
		r.Put("/{namespace}/{name}", gw.Update)
		r.Delete("/{namespace}/{name}", gw.Delete)
		r.Post("/{namespace}/{name}/deploy", gw.Deploy)
	})

	// HTTP Routes (namespace-aware)
	r.Route("/httproutes", func(r chi.Router) {
		r.Get("/", rt.List)
		r.Post("/", rt.Create)
		r.Get("/{namespace}/{name}", rt.Get)
		r.Put("/{namespace}/{name}", rt.Update)
		r.Delete("/{namespace}/{name}", rt.Delete)
		r.Post("/{namespace}/{name}/simulate", rt.Simulate)
	})

	// gRPC Routes
	r.Route("/grpcroutes", func(r chi.Router) {
		r.Get("/", rt.List)
		r.Post("/", rt.Create)
		r.Get("/{namespace}/{name}", rt.Get)
		r.Put("/{namespace}/{name}", rt.Update)
		r.Delete("/{namespace}/{name}", rt.Delete)
	})

	// TLS Routes
	r.Route("/tlsroutes", func(r chi.Router) {
		r.Get("/", rt.List)
		r.Post("/", rt.Create)
		r.Get("/{namespace}/{name}", rt.Get)
		r.Put("/{namespace}/{name}", rt.Update)
		r.Delete("/{namespace}/{name}", rt.Delete)
	})

	// TCP Routes
	r.Route("/tcproutes", func(r chi.Router) {
		r.Get("/", rt.List)
		r.Post("/", rt.Create)
		r.Get("/{namespace}/{name}", rt.Get)
		r.Put("/{namespace}/{name}", rt.Update)
		r.Delete("/{namespace}/{name}", rt.Delete)
	})

	// UDP Routes
	r.Route("/udproutes", func(r chi.Router) {
		r.Get("/", rt.List)
		r.Post("/", rt.Create)
		r.Get("/{namespace}/{name}", rt.Get)
		r.Put("/{namespace}/{name}", rt.Update)
		r.Delete("/{namespace}/{name}", rt.Delete)
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
			r.Get("/epp-decisions", infMet.EPPDecisions)
			r.Get("/ttft-histogram/{pool}", infMet.TTFTHistogram)
			r.Get("/tps-throughput/{pool}", infMet.TPSThroughput)
			r.Get("/queue-depth/{pool}", infMet.QueueDepthSeries)
			r.Get("/gpu-util/{pool}", infMet.GPUUtilSeries)
			r.Get("/kv-cache/{pool}", infMet.KVCacheSeries)
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
}
