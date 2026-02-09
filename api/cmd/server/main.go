package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/kubenetlabs/ngc/api/internal/server"
	"github.com/kubenetlabs/ngc/api/pkg/version"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server listen port")
	dbType := flag.String("db-type", "clickhouse", "Database backend type (clickhouse)")
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

	srv := server.New()

	addr := fmt.Sprintf(":%d", *port)
	if err := srv.Run(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
