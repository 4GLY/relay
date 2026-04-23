package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"relay/internal/relayapi"
	"relay/internal/services"
)

type Backend interface {
	Health(ctx context.Context) (relayapi.HealthResult, error)
	Capture(ctx context.Context, input services.CaptureInput) (services.CaptureResult, error)
	Promote(ctx context.Context, input services.PromoteInput) (services.PromoteResult, error)
	BuildPacket(ctx context.Context, input services.PacketBuildInput) (services.PacketBuildResult, error)
	Show(ctx context.Context, projectID string) (services.ShowResult, error)
	IssueAPIKey(ctx context.Context, input services.IssueAPIKeyInput) (services.IssueAPIKeyResult, error)
	ListAPIKeys(ctx context.Context) (services.ListAPIKeysResult, error)
	RevokeAPIKey(ctx context.Context, input services.RevokeAPIKeyInput) (services.RevokeAPIKeyResult, error)
	HasAdminToken() bool
	BaseURL() string
}

type Server struct {
	backend Backend
	server  *mcp.Server
}

func New(backend Backend) *Server {
	s := &Server{
		backend: backend,
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

	if s.backend.HasAdminToken() {
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
	result, err := s.backend.Health(ctx)
	if err != nil {
		return nil, healthOutput{}, err
	}
	return nil, healthOutput{
		Status:       result.Status,
		BaseURL:      s.clientBaseURL(),
		AdminEnabled: s.backend.HasAdminToken(),
	}, nil
}

type captureInput struct {
	Project        string                 `json:"project,omitempty" jsonschema:"Optional project name. When omitted, the server may infer the project from repo_path or use the bound project when it safely matches a project-scoped key"`
	Source         string                 `json:"source,omitempty" jsonschema:"Optional memory source such as chat, markdown, or manual. Defaults to manual"`
	Body           string                 `json:"body,omitempty" jsonschema:"Optional raw memory text to store. You can also supply note as an alias"`
	Note           string                 `json:"note,omitempty" jsonschema:"Alias for body. Optional raw memory text to store"`
	RepoPath       string                 `json:"repo_path,omitempty" jsonschema:"Optional repo path artifact to attach"`
	HandoffPath    string                 `json:"handoff_path,omitempty" jsonschema:"Optional handoff markdown path to attach"`
	DesignPath     string                 `json:"design_path,omitempty" jsonschema:"Optional design document path to attach"`
	ExtraArtifacts []captureArtifactInput `json:"extra_artifacts,omitempty" jsonschema:"Optional additional trusted artifacts to attach such as changed files, code paths, or PR diffs"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty" jsonschema:"Optional but recommended write key for safe retries"`
}

type captureArtifactInput struct {
	Type       string `json:"type" jsonschema:"Artifact type such as code_path, changed_files, or pr_diff"`
	SourcePath string `json:"source_path" jsonschema:"Local or canonical path for the artifact"`
	TrustLevel string `json:"trust_level,omitempty" jsonschema:"Optional trust level. Defaults to trusted"`
}

func (s *Server) captureTool(ctx context.Context, _ *mcp.CallToolRequest, input captureInput) (*mcp.CallToolResult, services.CaptureResult, error) {
	extraArtifacts := make([]services.CaptureArtifactInput, 0, len(input.ExtraArtifacts))
	for _, artifact := range input.ExtraArtifacts {
		extraArtifacts = append(extraArtifacts, services.CaptureArtifactInput{
			Type:       artifact.Type,
			SourcePath: artifact.SourcePath,
			TrustLevel: artifact.TrustLevel,
		})
	}
	result, err := s.backend.Capture(ctx, services.CaptureInput{
		Project:        input.Project,
		RepoPath:       input.RepoPath,
		HandoffPath:    input.HandoffPath,
		DesignPath:     input.DesignPath,
		ExtraArtifacts: extraArtifacts,
		Source:         input.Source,
		Body:           input.Body,
		Note:           input.Note,
		IdempotencyKey: input.IdempotencyKey,
	})
	return nil, result, err
}

type promoteInput struct {
	Project           string   `json:"project" jsonschema:"Required project name that owns the promoted memory"`
	Kind              string   `json:"kind" jsonschema:"Required promotion kind. Valid values: decision or question"`
	Summary           string   `json:"summary" jsonschema:"Required durable statement to preserve"`
	Reason            string   `json:"reason,omitempty" jsonschema:"Required when kind is decision. Omit for question when there is no settled answer yet"`
	SourceNoteIDs     []string `json:"source_note_ids,omitempty" jsonschema:"Optional supporting note ids"`
	SourceArtifactIDs []string `json:"source_artifact_ids,omitempty" jsonschema:"Optional supporting artifact ids"`
	IdempotencyKey    string   `json:"idempotency_key,omitempty" jsonschema:"Optional but recommended write key for safe retries"`
}

func (s *Server) promoteTool(ctx context.Context, _ *mcp.CallToolRequest, input promoteInput) (*mcp.CallToolResult, services.PromoteResult, error) {
	result, err := s.backend.Promote(ctx, services.PromoteInput{
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
	Project          string `json:"project" jsonschema:"Required project name to build the packet from"`
	Type             string `json:"type,omitempty" jsonschema:"Optional packet type. Defaults to resume"`
	Target           string `json:"target,omitempty" jsonschema:"Optional packet target. Defaults to codex in MCP flows"`
	Workflow         string `json:"workflow,omitempty" jsonschema:"Optional workflow selector for style-memory cues"`
	ArtifactType     string `json:"artifact_type,omitempty" jsonschema:"Optional artifact selector for style-memory cues"`
	TaskSummary      string `json:"task_summary,omitempty" jsonschema:"Optional current task summary to bind into the packet"`
	DisableStyleCues bool   `json:"disable_style_cues,omitempty" jsonschema:"When true, build the packet without approved style-memory cues"`
	PersistSnapshot  bool   `json:"persist_snapshot,omitempty" jsonschema:"When true, persist an immutable packet snapshot for deterministic replay"`
	IdempotencyKey   string `json:"idempotency_key,omitempty" jsonschema:"Optional but recommended key when persist_snapshot is true"`
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
	result, err := s.backend.BuildPacket(ctx, services.PacketBuildInput{
		Project:          input.Project,
		Type:             packetType,
		Target:           target,
		Workflow:         input.Workflow,
		ArtifactType:     input.ArtifactType,
		TaskSummary:      input.TaskSummary,
		DisableStyleCues: input.DisableStyleCues,
		PersistSnapshot:  input.PersistSnapshot,
		IdempotencyKey:   input.IdempotencyKey,
	})
	return nil, result, err
}

type showProjectInput struct {
	ProjectID string `json:"project_id" jsonschema:"Required canonical Relay project id, not the project name"`
}

func (s *Server) showProjectTool(ctx context.Context, _ *mcp.CallToolRequest, input showProjectInput) (*mcp.CallToolResult, services.ShowResult, error) {
	result, err := s.backend.Show(ctx, input.ProjectID)
	return nil, result, err
}

type issueAPIKeyInput struct {
	Name      string `json:"name" jsonschema:"Human-readable key name"`
	Scope     string `json:"scope,omitempty" jsonschema:"Optional key scope. Valid values: global or project. Defaults to global"`
	Project   string `json:"project,omitempty" jsonschema:"Optional project name for project-scoped keys"`
	ProjectID string `json:"project_id,omitempty" jsonschema:"Optional canonical project id for project-scoped keys"`
}

func (s *Server) issueAPIKeyTool(ctx context.Context, _ *mcp.CallToolRequest, input issueAPIKeyInput) (*mcp.CallToolResult, services.IssueAPIKeyResult, error) {
	result, err := s.backend.IssueAPIKey(ctx, services.IssueAPIKeyInput{
		Name:      input.Name,
		Scope:     input.Scope,
		Project:   input.Project,
		ProjectID: input.ProjectID,
	})
	return nil, result, err
}

type listAPIKeysInput struct{}

func (s *Server) listAPIKeysTool(ctx context.Context, _ *mcp.CallToolRequest, _ listAPIKeysInput) (*mcp.CallToolResult, services.ListAPIKeysResult, error) {
	result, err := s.backend.ListAPIKeys(ctx)
	return nil, result, err
}

type revokeAPIKeyInput struct {
	KeyID string `json:"key_id" jsonschema:"Canonical Relay api key id"`
}

func (s *Server) revokeAPIKeyTool(ctx context.Context, _ *mcp.CallToolRequest, input revokeAPIKeyInput) (*mcp.CallToolResult, services.RevokeAPIKeyResult, error) {
	result, err := s.backend.RevokeAPIKey(ctx, services.RevokeAPIKeyInput{KeyID: input.KeyID})
	return nil, result, err
}

func (s *Server) clientBaseURL() string {
	if s.backend == nil {
		return ""
	}
	return fmt.Sprintf("%s", s.backend.BaseURL())
}

type serviceBackend struct {
	service      services.Service
	baseURL      string
	adminEnabled bool
}

func NewFromService(service services.Service, baseURL string, adminEnabled bool) *Server {
	return New(serviceBackend{
		service:      service,
		baseURL:      baseURL,
		adminEnabled: adminEnabled,
	})
}

func (b serviceBackend) Health(_ context.Context) (relayapi.HealthResult, error) {
	return relayapi.HealthResult{Status: "ok"}, nil
}

func (b serviceBackend) Capture(ctx context.Context, input services.CaptureInput) (services.CaptureResult, error) {
	return b.service.Capture(ctx, input)
}

func (b serviceBackend) Promote(ctx context.Context, input services.PromoteInput) (services.PromoteResult, error) {
	return b.service.Promote(ctx, input)
}

func (b serviceBackend) BuildPacket(ctx context.Context, input services.PacketBuildInput) (services.PacketBuildResult, error) {
	return b.service.BuildPacket(ctx, input)
}

func (b serviceBackend) Show(ctx context.Context, projectID string) (services.ShowResult, error) {
	return b.service.Show(ctx, services.ShowInput{ProjectID: projectID})
}

func (b serviceBackend) IssueAPIKey(ctx context.Context, input services.IssueAPIKeyInput) (services.IssueAPIKeyResult, error) {
	return b.service.IssueAPIKey(b.adminContext(ctx), input)
}

func (b serviceBackend) ListAPIKeys(ctx context.Context) (services.ListAPIKeysResult, error) {
	return b.service.ListAPIKeys(b.adminContext(ctx))
}

func (b serviceBackend) RevokeAPIKey(ctx context.Context, input services.RevokeAPIKeyInput) (services.RevokeAPIKeyResult, error) {
	return b.service.RevokeAPIKey(b.adminContext(ctx), input)
}

func (b serviceBackend) HasAdminToken() bool {
	return b.adminEnabled
}

func (b serviceBackend) BaseURL() string {
	return b.baseURL
}

func (b serviceBackend) adminContext(ctx context.Context) context.Context {
	if !b.adminEnabled {
		return ctx
	}
	return services.ContextWithAuthInfo(ctx, services.AuthInfo{
		IsAdmin: true,
		Scope:   services.APIKeyScopeGlobal,
	})
}
