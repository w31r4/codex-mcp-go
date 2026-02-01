package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/w31r4/codex-mcp-go/internal/config"
	"github.com/w31r4/codex-mcp-go/internal/logging"
	server "github.com/w31r4/codex-mcp-go/internal/mcp"
)

func main() {
	configPath := flag.String("config", "", "Path to config file (optional). Can also be set via CODEX_MCP_CONFIG.")
	flag.Parse()

	path := strings.TrimSpace(*configPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("CODEX_MCP_CONFIG"))
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := logging.New(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logging.SetGlobalLogger(logger)
	logger.Info("starting mcp server", "server_name", cfg.Server.Name, "server_version", cfg.Server.Version)

	if err := server.Run(context.Background(), cfg); err != nil {
		logger.Error("server stopped with error", "error", err.Error())
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}
