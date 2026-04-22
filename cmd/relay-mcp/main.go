package main

import (
	"context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"log"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/mcpserver"
)

func main() {
	cfg := config.Load()
	runtime, err := app.NewRuntime(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}

	server := mcpserver.NewFromService(runtime.Services, cfg.BaseURL, adminToolsEnabled(cfg))
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func adminToolsEnabled(cfg config.Config) bool {
	return cfg.AdminToken != ""
}
