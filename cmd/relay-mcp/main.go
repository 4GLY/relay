package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"relay/internal/mcpserver"
	"relay/internal/relayapi"
)

func main() {
	client := relayapi.NewClient(relayapi.LoadConfigFromEnv())
	server := mcpserver.New(client)

	transport := strings.ToLower(strings.TrimSpace(os.Getenv("RELAY_MCP_TRANSPORT")))
	if transport == "" || transport == "stdio" {
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
		return
	}

	if transport != "http" {
		log.Fatalf("unsupported RELAY_MCP_TRANSPORT %q", transport)
	}

	addr := os.Getenv("RELAY_MCP_ADDR")
	if addr == "" {
		addr = ":8091"
	}

	path := os.Getenv("RELAY_MCP_PATH")
	if path == "" {
		path = "/mcp"
	}

	mcpToken := os.Getenv("RELAY_MCP_TOKEN")
	streamable := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server.Server()
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"transport":"http-mcp"}`))
	})
	mux.Handle(path, requireBearerToken(mcpToken, streamable))

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("relay-mcp http listening on %s%s", addr, path)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func requireBearerToken(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return auth.RequireBearerToken(func(_ context.Context, provided string, _ *http.Request) (*auth.TokenInfo, error) {
		if strings.TrimSpace(provided) == token {
			return &auth.TokenInfo{Expiration: time.Now().Add(24 * time.Hour)}, nil
		}
		return nil, auth.ErrInvalidToken
	}, nil)(next)
}
