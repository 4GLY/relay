package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type staticBearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t *staticBearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(cloned)
}

func newAuthedHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: &staticBearerTransport{
			token: token,
			base:  http.DefaultTransport,
		},
	}
}

func main() {
	baseURL := os.Getenv("RELAY_MCP_URL")
	if baseURL == "" {
		baseURL = "https://relay.4gly.dev/mcp"
	}

	token := os.Getenv("RELAY_CLIENT_TOKEN")
	if token == "" {
		token = os.Getenv("RELAY_MCP_TOKEN")
	}
	if token == "" {
		log.Fatal("RELAY_CLIENT_TOKEN or RELAY_MCP_TOKEN is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "relay-go-example",
		Version: "v0.0.1",
	}, nil)

	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   baseURL,
		HTTPClient: newAuthedHTTPClient(token),
	}, nil)
	if err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("tools/list failed: %v", err)
	}

	health, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "relay_health",
		Arguments: map[string]any{},
	})
	if err != nil {
		log.Fatalf("relay_health failed: %v", err)
	}

	out := map[string]any{
		"tool_count": len(tools.Tools),
		"tool_names": toolNames(tools.Tools),
		"health":     health.StructuredContent,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("encode failed: %v", err)
	}
}

func toolNames(tools []*mcp.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		names = append(names, tool.Name)
	}
	return names
}
