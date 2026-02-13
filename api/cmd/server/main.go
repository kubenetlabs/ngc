package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"time"

	"github.com/kubenetlabs/ngc/api/internal/alerting"
	chprovider "github.com/kubenetlabs/ngc/api/internal/clickhouse"
	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/database"
	"github.com/kubenetlabs/ngc/api/internal/inference"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
	mc "github.com/kubenetlabs/ngc/api/internal/multicluster"
	prom "github.com/kubenetlabs/ngc/api/internal/prometheus"
	"github.com/kubenetlabs/ngc/api/internal/server"
	"github.com/kubenetlabs/ngc/api/pkg/version"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server listen port")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file (optional, defaults to in-cluster or ~/.kube/config)")
	clustersConfig := flag.String("clusters-config", "", "Path to clusters YAML config (enables multi-cluster)")
	dbType := flag.String("db-type", "mock", "Metrics provider backend (mock, clickhouse)")
	clickhouseURL := flag.String("clickhouse-url", "localhost:9000", "ClickHouse connection URL")
	prometheusURL := flag.String("prometheus-url", "", "Prometheus server URL (e.g., http://prometheus:9090)")
	configDB := flag.String("config-db", "ngf-console.db", "Path to SQLite config database")
	alertWebhooks := flag.String("alert-webhooks", "", "Comma-separated webhook URLs for alert notifications")
	multicluster := flag.Bool("multicluster", false, "Enable CRD-based multi-cluster mode (reads ManagedCluster CRDs)")
	multiclusterNS := flag.String("multicluster-namespace", "ngf-system", "Namespace for ManagedCluster CRDs")
	multiclusterDefault := flag.String("multicluster-default", "", "Default cluster name in multi-cluster mode")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ngc-api %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
		os.Exit(0)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting NGF Console API server",
		"port", *port,
		"db_type", *dbType,
		"clickhouse_url", *clickhouseURL,
		"version", version.Version,
	)

	var mgr cluster.Provider
	var pool *mc.ClientPool
	if *multicluster {
		// CRD-based multi-cluster mode: read ManagedCluster CRDs from hub.
		k8sClient, err := kubernetes.New(*kubeconfig)
		if err != nil {
			slog.Error("failed to create hub kubernetes client", "error", err)
			os.Exit(1)
		}
		pool = mc.NewClientPool(k8sClient.DynamicClient(), *multiclusterNS)
		if err := pool.Sync(context.Background()); err != nil {
			slog.Error("failed to sync cluster pool", "error", err)
			os.Exit(1)
		}
		defaultName := *multiclusterDefault
		if defaultName == "" {
			names := pool.Names()
			if len(names) > 0 {
				defaultName = names[0]
			}
		}
		mgr = mc.NewPoolAdapter(pool, defaultName)
		slog.Info("CRD-based multi-cluster mode enabled", "clusters", pool.Names(), "namespace", *multiclusterNS)

		// Start health checker.
		go mc.RunHealthChecker(context.Background(), pool, 30*time.Second)
	} else if *clustersConfig != "" {
		cfg, err := cluster.LoadConfig(*clustersConfig)
		if err != nil {
			slog.Error("failed to load clusters config", "error", err)
			os.Exit(1)
		}
		fileMgr, err := cluster.New(cfg)
		if err != nil {
			slog.Error("failed to create cluster manager", "error", err)
			os.Exit(1)
		}
		mgr = fileMgr
		slog.Info("file-based multi-cluster mode enabled", "clusters", fileMgr.Names())
	} else {
		k8sClient, err := kubernetes.New(*kubeconfig)
		if err != nil {
			slog.Error("failed to create kubernetes client", "error", err)
			os.Exit(1)
		}
		mgr = cluster.NewSingleCluster(k8sClient)
		slog.Info("single-cluster mode")
	}

	// Initialize metrics provider (mock for dev, ClickHouse for prod)
	var metricsProvider inference.MetricsProvider
	var chClient *chprovider.Client
	if *dbType == "clickhouse" && *clickhouseURL != "" {
		var err error
		chClient, err = chprovider.New(*clickhouseURL)
		if err != nil {
			slog.Error("failed to create clickhouse client (explicitly configured)", "error", err)
			os.Exit(1)
		}
		metricsProvider = chprovider.NewProvider(chClient)
		slog.Info("using clickhouse metrics provider", "url", *clickhouseURL)
	} else {
		metricsProvider = inference.NewMockProvider()
		slog.Info("using mock metrics provider")
	}

	// Start inference pool sync loop (updates ClickHouse pool status from CRDs).
	if pool != nil && metricsProvider != nil {
		go inference.RunSyncLoop(context.Background(), pool, metricsProvider, 30*time.Second)
	}

	// Start metrics scraper (scrapes vLLM /metrics → ClickHouse).
	if pool != nil && chClient != nil {
		go inference.RunMetricsScraper(context.Background(), pool, chClient.Conn(), metricsProvider, 15*time.Second)
	}

	// Initialize Prometheus client if configured.
	var promClient *prom.Client
	if *prometheusURL != "" {
		var err error
		promClient, err = prom.New(*prometheusURL)
		if err != nil {
			slog.Error("failed to create prometheus client", "error", err, "url", *prometheusURL)
			os.Exit(1)
		}
		slog.Info("prometheus client configured", "url", *prometheusURL)
	} else {
		slog.Info("prometheus not configured, RED metrics endpoints will return 503")
	}

	// Initialize config database (SQLite).
	store, err := database.NewSQLite(*configDB)
	if err != nil {
		slog.Error("failed to open config database", "error", err, "path", *configDB)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		slog.Error("failed to migrate config database", "error", err)
		os.Exit(1)
	}
	slog.Info("config database ready", "path", *configDB)

	// Parse alert webhook URLs.
	var webhooks []alerting.WebhookConfig
	if *alertWebhooks != "" {
		for _, url := range strings.Split(*alertWebhooks, ",") {
			url = strings.TrimSpace(url)
			if url != "" {
				webhooks = append(webhooks, alerting.WebhookConfig{URL: url})
			}
		}
		slog.Info("alert webhooks configured", "count", len(webhooks))
	}

	srv := server.New(server.Config{
		ClusterManager:  mgr,
		MetricsProvider: metricsProvider,
		Store:           store,
		PromClient:      promClient,
		CHClient:        chClient,
		Webhooks:        webhooks,
		Pool:            pool,
	})

	addr := fmt.Sprintf(":%d", *port)
	if err := srv.Run(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
