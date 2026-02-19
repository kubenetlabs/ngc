package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/kubenetlabs/ngc/api/internal/alerting"
	ch "github.com/kubenetlabs/ngc/api/internal/clickhouse"
	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/database"
	"github.com/kubenetlabs/ngc/api/internal/handlers"
	"github.com/kubenetlabs/ngc/api/internal/inference"
	mc "github.com/kubenetlabs/ngc/api/internal/multicluster"
	prom "github.com/kubenetlabs/ngc/api/internal/prometheus"
)

// Config holds server dependencies.
type Config struct {
	ClusterManager  cluster.Provider
	MetricsProvider inference.MetricsProvider
	Store           database.Store
	PromClient      *prom.Client
	CHClient        *ch.Client
	Webhooks        []alerting.WebhookConfig
	Pool            *mc.ClientPool // non-nil when using CRD-based multi-cluster
}

// Server is the main HTTP server for the NGF Console API.
type Server struct {
	Router     chi.Router
	Config     Config
	Hub        *Hub
	Evaluator  *alerting.Evaluator
	AuthConfig AuthConfig
}

// New creates a new Server with all routes and middleware configured.
func New(cfg Config) *Server {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(RequestLogger)
	r.Use(CORSMiddleware)
	r.Use(chimw.Recoverer)
	r.Use(MaxBodySize(1 << 20)) // 1MB max body size

	hub := NewHub()
	RegisterInferenceTopics(hub, cfg.MetricsProvider, cfg.ClusterManager)
	hub.Start()

	// Create and start the alert evaluator.
	eval := alerting.New(cfg.Store, cfg.PromClient, cfg.Webhooks)
	eval.Start(context.Background())

	authCfg := AuthConfig{
		Enabled:   os.Getenv("AUTH_ENABLED") == "true",
		JWTSecret: os.Getenv("JWT_SECRET"),
		Issuer:    os.Getenv("JWT_ISSUER"),
	}
	if authCfg.Enabled {
		slog.Info("JWT authentication enabled")
	}

	s := &Server{Router: r, Config: cfg, Hub: hub, Evaluator: eval, AuthConfig: authCfg}
	s.registerRoutes()

	return s
}

// Run starts the HTTP server on the given address with graceful shutdown.
func (s *Server) Run(addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.Router,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		slog.Info("shutting down server", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	slog.Info("server stopped gracefully")
	return nil
}

// registerRoutes mounts all API v1 route groups.
func (s *Server) registerRoutes() {
	gw := &handlers.GatewayHandler{}
	rt := &handlers.RouteHandler{}
	grpcRt := &handlers.GRPCRouteHandler{}
	tlsRt := &handlers.TLSRouteHandler{}
	tcpRt := &handlers.TCPRouteHandler{}
	udpRt := &handlers.UDPRouteHandler{}
	cfgHandler := &handlers.ConfigHandler{}
	clusterHandler := &handlers.ClusterHandler{Manager: s.Config.ClusterManager, Pool: s.Config.Pool}
	pol := &handlers.PolicyHandler{}
	cert := &handlers.CertificateHandler{}
	met := &handlers.MetricsHandler{Prom: s.Config.PromClient}
	lg := &handlers.LogHandler{CH: s.Config.CHClient}
	topo := &handlers.TopologyHandler{}
	diag := &handlers.DiagnosticsHandler{}
	inf := &handlers.InferenceHandler{Provider: s.Config.MetricsProvider}
	infMet := &handlers.InferenceMetricsHandler{Provider: s.Config.MetricsProvider}
	var chProvider *ch.Provider
	if s.Config.CHClient != nil {
		chProvider = ch.NewProviderFromClient(s.Config.CHClient)
	}
	infDiag := &handlers.InferenceDiagHandler{CHClient: s.Config.CHClient, Provider: chProvider}
	infStack := &handlers.InferenceStackHandler{MetricsProvider: s.Config.MetricsProvider}
	gwBundle := &handlers.GatewayBundleHandler{}
	coex := &handlers.CoexistenceHandler{}
	xc := &handlers.XCHandler{Prom: s.Config.PromClient}
	mig := &handlers.MigrationHandler{}
	aud := &handlers.AuditHandler{Store: s.Config.Store}
	alert := &handlers.AlertHandler{Store: s.Config.Store, Evaluator: s.Evaluator}

	globalHandler := &handlers.GlobalHandler{Pool: s.Config.Pool, Manager: s.Config.ClusterManager}

	// Health check endpoint (outside /api/v1 for simplicity with probes)
	s.Router.Get("/api/v1/health", handlers.HealthCheck)

	s.Router.Route("/api/v1", func(r chi.Router) {
		// Auth middleware (no-op when AUTH_ENABLED is not "true")
		r.Use(AuthMiddleware(s.AuthConfig))

		// Cluster management (hub-level, no cluster middleware)
		r.Get("/clusters", clusterHandler.List)
		r.Post("/clusters", clusterHandler.Register)
		r.Get("/clusters/summary", clusterHandler.Summary)

		// Global cross-cluster aggregation endpoints
		r.Route("/global", func(r chi.Router) {
			r.Get("/gateways", globalHandler.Gateways)
			r.Get("/routes", globalHandler.Routes)
			r.Get("/gpu-capacity", globalHandler.GPUCapacity)
		})

		// Cluster-specific management + resource routes
		r.Route("/clusters/{cluster}", func(r chi.Router) {
			// Management endpoints (matched by specific sub-paths first)
			r.Get("/detail", clusterHandler.Get)
			r.Delete("/", clusterHandler.Unregister)
			r.Post("/test", clusterHandler.TestConnection)
			r.Post("/install-agent", clusterHandler.InstallAgent)
			r.Post("/heartbeat", clusterHandler.Heartbeat)

			// Cluster-scoped resource routes
			r.Group(func(r chi.Router) {
				r.Use(ClusterResolver(s.Config.ClusterManager))
				s.mountResourceRoutes(r, gw, rt, grpcRt, tlsRt, tcpRt, udpRt, cfgHandler, pol, cert, met, lg, topo, diag, inf, infMet, infDiag, infStack, gwBundle, coex, xc, mig, aud, alert)
			})
		})

		// Legacy routes (backward compat — uses default cluster)
		r.Group(func(r chi.Router) {
			r.Use(ClusterResolver(s.Config.ClusterManager))
			s.mountResourceRoutes(r, gw, rt, grpcRt, tlsRt, tcpRt, udpRt, cfgHandler, pol, cert, met, lg, topo, diag, inf, infMet, infDiag, infStack, gwBundle, coex, xc, mig, aud, alert)
		})

		// WebSocket (Hub-integrated, supports ?topic= query param)
		r.Get("/ws", HandleLegacyWS(s.Hub))

		// Events (alias for WebSocket)
		r.Get("/events", HandleLegacyWS(s.Hub))

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
	grpcRt *handlers.GRPCRouteHandler,
	tlsRt *handlers.TLSRouteHandler,
	tcpRt *handlers.TCPRouteHandler,
	udpRt *handlers.UDPRouteHandler,
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
	infStack *handlers.InferenceStackHandler,
	gwBundle *handlers.GatewayBundleHandler,
	coex *handlers.CoexistenceHandler,
	xc *handlers.XCHandler,
	mig *handlers.MigrationHandler,
	aud *handlers.AuditHandler,
	alert *handlers.AlertHandler,
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

	// GatewayBundles (CRD-backed via dynamic client)
	r.Route("/gatewaybundles", func(r chi.Router) {
		r.Get("/", gwBundle.List)
		r.Post("/", gwBundle.Create)
		r.Get("/{namespace}/{name}", gwBundle.Get)
		r.Put("/{namespace}/{name}", gwBundle.Update)
		r.Delete("/{namespace}/{name}", gwBundle.Delete)
		r.Get("/{namespace}/{name}/status", gwBundle.GetStatus)
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

	// gRPC Routes (namespace-aware)
	r.Route("/grpcroutes", func(r chi.Router) {
		r.Get("/", grpcRt.List)
		r.Post("/", grpcRt.Create)
		r.Get("/{namespace}/{name}", grpcRt.Get)
		r.Put("/{namespace}/{name}", grpcRt.Update)
		r.Delete("/{namespace}/{name}", grpcRt.Delete)
	})

	// TLS Routes (namespace-aware)
	r.Route("/tlsroutes", func(r chi.Router) {
		r.Get("/", tlsRt.List)
		r.Post("/", tlsRt.Create)
		r.Get("/{namespace}/{name}", tlsRt.Get)
		r.Put("/{namespace}/{name}", tlsRt.Update)
		r.Delete("/{namespace}/{name}", tlsRt.Delete)
	})

	// TCP Routes (namespace-aware)
	r.Route("/tcproutes", func(r chi.Router) {
		r.Get("/", tcpRt.List)
		r.Post("/", tcpRt.Create)
		r.Get("/{namespace}/{name}", tcpRt.Get)
		r.Put("/{namespace}/{name}", tcpRt.Update)
		r.Delete("/{namespace}/{name}", tcpRt.Delete)
	})

	// UDP Routes (namespace-aware)
	r.Route("/udproutes", func(r chi.Router) {
		r.Get("/", udpRt.List)
		r.Post("/", udpRt.Create)
		r.Get("/{namespace}/{name}", udpRt.Get)
		r.Put("/{namespace}/{name}", udpRt.Update)
		r.Delete("/{namespace}/{name}", udpRt.Delete)
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
			r.Get("/active-requests/{pool}", infMet.ActiveRequestsSeries)
			r.Get("/gpu-util/{pool}", infMet.GPUUtilSeries)
			r.Get("/kv-cache/{pool}", infMet.KVCacheSeries)
		})

		// Inference Diagnostics
		r.Route("/diagnostics", func(r chi.Router) {
			r.Get("/slow", infDiag.SlowInference)
			r.Post("/replay", infDiag.Replay)
			r.Post("/benchmark", infDiag.Benchmark)
		})

		// InferenceStacks (CRD-backed via dynamic client)
		r.Route("/stacks", func(r chi.Router) {
			r.Get("/", infStack.List)
			r.Post("/", infStack.Create)
			r.Get("/{namespace}/{name}", infStack.Get)
			r.Put("/{namespace}/{name}", infStack.Update)
			r.Delete("/{namespace}/{name}", infStack.Delete)
			r.Get("/{namespace}/{name}/status", infStack.GetStatus)
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

	// Alerts
	r.Route("/alerts", func(r chi.Router) {
		r.Get("/", alert.List)
		r.Post("/", alert.Create)
		r.Get("/firing", alert.Firing)
		r.Get("/{id}", alert.Get)
		r.Put("/{id}", alert.Update)
		r.Delete("/{id}", alert.Delete)
		r.Post("/{id}/toggle", alert.Toggle)
	})
}
