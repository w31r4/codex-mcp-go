package main

import (
	"context"
	"fmt"
	"os"

	server "github.com/w31r4/codex-mcp-go/internal/mcp"
)

func main() {
	if err := server.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}
