package main

import (
	"context"
	"fmt"
	"os"

	"github.com/w31r4/codex-mcp-go/internal/logging"
	server "github.com/w31r4/codex-mcp-go/internal/mcp"
)

func main() {
	logger, err := logging.New(logging.DefaultConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logging.SetGlobalLogger(logger)
	logger.Info("starting mcp server")

	if err := server.Run(context.Background()); err != nil {
		logger.Error("server stopped with error", "error", err.Error())
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}
