package main

import (
	"context"
	"fmt"
	"os"

	server "codex4kilomcp/internal/mcp"
)

func main() {
	if err := server.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}
