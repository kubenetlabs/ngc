package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	chprovider "github.com/kubenetlabs/ngc/api/internal/clickhouse"
	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/inference"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
	"github.com/kubenetlabs/ngc/api/internal/server"
	"github.com/kubenetlabs/ngc/api/pkg/version"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server listen port")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file (optional, defaults to in-cluster or ~/.kube/config)")
	clustersConfig := flag.String("clusters-config", "", "Path to clusters YAML config (enables multi-cluster)")
	dbType := flag.String("db-type", "mock", "Metrics provider backend (mock, clickhouse)")
	clickhouseURL := flag.String("clickhouse-url", "localhost:9000", "ClickHouse connection URL")
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

	var mgr *cluster.Manager
	if *clustersConfig != "" {
		cfg, err := cluster.LoadConfig(*clustersConfig)
		if err != nil {
			slog.Error("failed to load clusters config", "error", err)
			os.Exit(1)
		}
		mgr, err = cluster.New(cfg)
		if err != nil {
			slog.Error("failed to create cluster manager", "error", err)
			os.Exit(1)
		}
		slog.Info("multi-cluster mode enabled", "clusters", mgr.Names())
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
	if *dbType == "clickhouse" && *clickhouseURL != "" {
		chClient, err := chprovider.New(*clickhouseURL)
		if err != nil {
			slog.Warn("failed to create clickhouse client, falling back to mock", "error", err)
			metricsProvider = inference.NewMockProvider()
		} else {
			metricsProvider = chprovider.NewProvider(chClient)
			slog.Info("using clickhouse metrics provider", "url", *clickhouseURL)
		}
	} else {
		metricsProvider = inference.NewMockProvider()
		slog.Info("using mock metrics provider")
	}

	srv := server.New(server.Config{
		ClusterManager:  mgr,
		MetricsProvider: metricsProvider,
	})

	addr := fmt.Sprintf(":%d", *port)
	if err := srv.Run(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
