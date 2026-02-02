package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/w31r4/codex-mcp-go/internal/config"
	"github.com/w31r4/codex-mcp-go/internal/logging"
	server "github.com/w31r4/codex-mcp-go/internal/mcp"
)

func main() {
	configPath := flag.String("config", "", "Path to config file (optional). Can also be set via CODEX_MCP_CONFIG.")
	safeLocal := flag.Bool("safe-local", false, "Enable safer defaults for local usage (read-only default sandbox, disable yolo, restrict work dirs to $HOME unless overridden). Can also be set via CODEX_SAFE_LOCAL=true.")
	safeLocalRoot := flag.String("safe-local-root", "", "Comma-separated allowed workdir prefixes when --safe-local is enabled. Can also be set via CODEX_SAFE_LOCAL_ROOT.")
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

	enableSafeLocal := *safeLocal
	if !enableSafeLocal {
		if v := strings.TrimSpace(os.Getenv("CODEX_SAFE_LOCAL")); v != "" {
			if b, parseErr := strconv.ParseBool(v); parseErr == nil {
				enableSafeLocal = b
			}
		}
	}
	root := strings.TrimSpace(*safeLocalRoot)
	if root == "" {
		root = strings.TrimSpace(os.Getenv("CODEX_SAFE_LOCAL_ROOT"))
	}
	if enableSafeLocal {
		if err := config.ApplySafeLocalPreset(cfg, root); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to apply safe-local preset: %v\n", err)
			os.Exit(1)
		}
	}

	logger, err := logging.New(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logging.SetGlobalLogger(logger)
	logger.Info("starting mcp server", "server_name", cfg.Server.Name, "server_version", cfg.Server.Version)
	if enableSafeLocal {
		logger.Info("safe-local preset enabled", "allowed_work_dirs", cfg.Security.AllowedWorkDirs, "disable_yolo", cfg.Security.DisableYolo, "default_sandbox", cfg.Security.DefaultSandbox)
	}

	if err := server.Run(context.Background(), cfg); err != nil {
		logger.Error("server stopped with error", "error", err.Error())
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}
