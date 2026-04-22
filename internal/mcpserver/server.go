package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"relay/internal/relayapi"
	"relay/internal/services"
)

type Server struct {
	client *relayapi.Client
	server *mcp.Server
}

func New(client *relayapi.Client) *Server {
	s := &Server{
		client: client,
		server: mcp.NewServer(&mcp.Implementation{
			Name:    "relay-mcp",
			Version: "v1.0.0",
		}, nil),
	}
	s.registerTools()
	return s
}

func (s *Server) Run(ctx context.Context, transport mcp.Transport) error {
	return s.server.Run(ctx, transport)
}

func (s *Server) Server() *mcp.Server {
	return s.server
}

func (s *Server) registerTools() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "relay_health",
		Title:       "Relay Health",
		Description: "Check Relay API reachability and auth readiness.",
	}, s.healthTool)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "relay_capture",
		Title:       "Relay Capture",
		Description: "Store raw project memory and optional artifact paths in Relay.",
	}, s.captureTool)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "relay_promote",
		Title:       "Relay Promote",
		Description: "Promote stored memory into a durable decision or open question.",
	}, s.promoteTool)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "relay_build_packet",
		Title:       "Relay Build Packet",
		Description: "Build an agent-ready packet from stored Relay memory.",
	}, s.buildPacketTool)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "relay_show_project",
		Title:       "Relay Show Project",
		Description: "Inspect aggregate Relay project state by canonical project id.",
	}, s.showProjectTool)

	if s.client.HasAdminToken() {
		mcp.AddTool(s.server, &mcp.Tool{
			Name:        "relay_issue_api_key",
			Title:       "Relay Issue API Key",
			Description: "Mint a new Relay API key for an agent or client.",
		}, s.issueAPIKeyTool)
		mcp.AddTool(s.server, &mcp.Tool{
			Name:        "relay_list_api_keys",
			Title:       "Relay List API Keys",
			Description: "List issued Relay API keys and their revocation state.",
		}, s.listAPIKeysTool)
		mcp.AddTool(s.server, &mcp.Tool{
			Name:        "relay_revoke_api_key",
			Title:       "Relay Revoke API Key",
			Description: "Revoke a previously issued Relay API key.",
		}, s.revokeAPIKeyTool)
	}
}

type healthInput struct{}

type healthOutput struct {
	Status       string `json:"status" jsonschema:"Relay health status"`
	BaseURL      string `json:"base_url" jsonschema:"Resolved Relay API base URL"`
	AdminEnabled bool   `json:"admin_enabled" jsonschema:"Whether admin tools are enabled in this MCP server"`
}

func (s *Server) healthTool(ctx context.Context, _ *mcp.CallToolRequest, _ healthInput) (*mcp.CallToolResult, healthOutput, error) {
	result, err := s.client.Health(ctx)
	if err != nil {
		return nil, healthOutput{}, err
	}
	return nil, healthOutput{
		Status:       result.Status,
		BaseURL:      s.clientBaseURL(),
		AdminEnabled: s.client.HasAdminToken(),
	}, nil
}

type captureInput struct {
	Project        string `json:"project" jsonschema:"Project name to attach the captured memory to"`
	RepoPath       string `json:"repo_path,omitempty" jsonschema:"Optional repo path artifact to attach"`
	HandoffPath    string `json:"handoff_path,omitempty" jsonschema:"Optional handoff document path to attach"`
	DesignPath     string `json:"design_path,omitempty" jsonschema:"Optional design document path to attach"`
	Note           string `json:"note,omitempty" jsonschema:"Optional short note field"`
	Source         string `json:"source" jsonschema:"Memory source, such as chat or markdown"`
	Body           string `json:"body" jsonschema:"Raw memory text to store"`
	IdempotencyKey string `json:"idempotency_key,omitempty" jsonschema:"Optional idempotency key for safe retries"`
}

func (s *Server) captureTool(ctx context.Context, _ *mcp.CallToolRequest, input captureInput) (*mcp.CallToolResult, services.CaptureResult, error) {
	result, err := s.client.Capture(ctx, services.CaptureInput{
		Project:        input.Project,
		RepoPath:       input.RepoPath,
		HandoffPath:    input.HandoffPath,
		DesignPath:     input.DesignPath,
		Note:           input.Note,
		Source:         input.Source,
		Body:           input.Body,
		IdempotencyKey: input.IdempotencyKey,
	})
	return nil, result, err
}

type promoteInput struct {
	Project           string   `json:"project" jsonschema:"Project name that owns the promoted memory"`
	Kind              string   `json:"kind" jsonschema:"Promotion kind: decision or question"`
	Summary           string   `json:"summary" jsonschema:"Durable statement to preserve"`
	Reason            string   `json:"reason,omitempty" jsonschema:"Why this decision or question matters"`
	SourceNoteIDs     []string `json:"source_note_ids,omitempty" jsonschema:"Supporting note ids"`
	SourceArtifactIDs []string `json:"source_artifact_ids,omitempty" jsonschema:"Supporting artifact ids"`
	IdempotencyKey    string   `json:"idempotency_key,omitempty" jsonschema:"Optional idempotency key for safe retries"`
}

func (s *Server) promoteTool(ctx context.Context, _ *mcp.CallToolRequest, input promoteInput) (*mcp.CallToolResult, services.PromoteResult, error) {
	result, err := s.client.Promote(ctx, services.PromoteInput{
		Project:           input.Project,
		Kind:              input.Kind,
		Summary:           input.Summary,
		Reason:            input.Reason,
		SourceNoteIDs:     input.SourceNoteIDs,
		SourceArtifactIDs: input.SourceArtifactIDs,
		IdempotencyKey:    input.IdempotencyKey,
	})
	return nil, result, err
}

type buildPacketInput struct {
	Project string `json:"project" jsonschema:"Project name to build the packet from"`
	Type    string `json:"type,omitempty" jsonschema:"Packet type, usually resume"`
	Target  string `json:"target,omitempty" jsonschema:"Target agent or client, such as codex"`
}

func (s *Server) buildPacketTool(ctx context.Context, _ *mcp.CallToolRequest, input buildPacketInput) (*mcp.CallToolResult, services.PacketBuildResult, error) {
	packetType := input.Type
	if packetType == "" {
		packetType = "resume"
	}
	target := input.Target
	if target == "" {
		target = "codex"
	}
	result, err := s.client.BuildPacket(ctx, services.PacketBuildInput{
		Project: input.Project,
		Type:    packetType,
		Target:  target,
	})
	return nil, result, err
}

type showProjectInput struct {
	ProjectID string `json:"project_id" jsonschema:"Canonical Relay project id"`
}

func (s *Server) showProjectTool(ctx context.Context, _ *mcp.CallToolRequest, input showProjectInput) (*mcp.CallToolResult, services.ShowResult, error) {
	result, err := s.client.Show(ctx, input.ProjectID)
	return nil, result, err
}

type issueAPIKeyInput struct {
	Name string `json:"name" jsonschema:"Human-readable key name"`
}

func (s *Server) issueAPIKeyTool(ctx context.Context, _ *mcp.CallToolRequest, input issueAPIKeyInput) (*mcp.CallToolResult, services.IssueAPIKeyResult, error) {
	result, err := s.client.IssueAPIKey(ctx, services.IssueAPIKeyInput{Name: input.Name})
	return nil, result, err
}

type listAPIKeysInput struct{}

func (s *Server) listAPIKeysTool(ctx context.Context, _ *mcp.CallToolRequest, _ listAPIKeysInput) (*mcp.CallToolResult, services.ListAPIKeysResult, error) {
	result, err := s.client.ListAPIKeys(ctx)
	return nil, result, err
}

type revokeAPIKeyInput struct {
	KeyID string `json:"key_id" jsonschema:"Canonical Relay api key id"`
}

func (s *Server) revokeAPIKeyTool(ctx context.Context, _ *mcp.CallToolRequest, input revokeAPIKeyInput) (*mcp.CallToolResult, services.RevokeAPIKeyResult, error) {
	result, err := s.client.RevokeAPIKey(ctx, services.RevokeAPIKeyInput{KeyID: input.KeyID})
	return nil, result, err
}

func (s *Server) clientBaseURL() string {
	if s.client == nil {
		return ""
	}
	return fmt.Sprintf("%s", s.client.BaseURL())
}
