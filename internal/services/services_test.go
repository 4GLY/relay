package services

import (
	"context"
	"strings"
	"testing"

	"relay/internal/domain"
	"relay/internal/lib"
)

type fakeProjectStore struct {
	projects map[string]domain.Project
}

func (s *fakeProjectStore) EnsureProject(_ context.Context, project domain.Project) (domain.Project, error) {
	if s.projects == nil {
		s.projects = map[string]domain.Project{}
	}
	s.projects[project.Name] = project
	return project, nil
}

func (s *fakeProjectStore) GetByName(_ context.Context, name string) (domain.Project, error) {
	project, ok := s.projects[name]
	if !ok {
		return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
	}
	return project, nil
}

func (s *fakeProjectStore) GetByID(_ context.Context, id string) (domain.Project, error) {
	for _, project := range s.projects {
		if project.ID == id {
			return project, nil
		}
	}
	return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
}

type fakeNoteStore struct {
	items []domain.Note
}

func (s *fakeNoteStore) CreateNote(_ context.Context, note domain.Note) (domain.Note, error) {
	s.items = append(s.items, note)
	return note, nil
}

func (s *fakeNoteStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (s *fakeNoteStore) ListByProject(_ context.Context, projectID string) ([]domain.Note, error) {
	var items []domain.Note
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeArtifactStore struct {
	items []domain.Artifact
}

func (s *fakeArtifactStore) CreateArtifact(_ context.Context, artifact domain.Artifact) (domain.Artifact, error) {
	s.items = append(s.items, artifact)
	return artifact, nil
}

func (s *fakeArtifactStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (s *fakeArtifactStore) ListByProject(_ context.Context, projectID string) ([]domain.Artifact, error) {
	var items []domain.Artifact
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeDecisionStore struct {
	items []domain.Decision
}

func (s *fakeDecisionStore) CreateDecision(_ context.Context, decision domain.Decision) (domain.Decision, error) {
	s.items = append(s.items, decision)
	return decision, nil
}

func (s *fakeDecisionStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (s *fakeDecisionStore) ListByProject(_ context.Context, projectID string) ([]domain.Decision, error) {
	var items []domain.Decision
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeOpenQuestionStore struct {
	items []domain.OpenQuestion
}

func (s *fakeOpenQuestionStore) CreateOpenQuestion(_ context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error) {
	s.items = append(s.items, question)
	return question, nil
}

func (s *fakeOpenQuestionStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (s *fakeOpenQuestionStore) ListByProject(_ context.Context, projectID string) ([]domain.OpenQuestion, error) {
	var items []domain.OpenQuestion
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakePacketStore struct {
	latest domain.Packet
}

func (s *fakePacketStore) CreatePacket(_ context.Context, packet domain.Packet) (domain.Packet, error) {
	s.latest = packet
	return packet, nil
}

func (s *fakePacketStore) LatestByProject(_ context.Context, _ string) (domain.Packet, error) {
	return s.latest, nil
}

type fakeAPIKeyStore struct {
	itemsByHash map[string]domain.APIKey
	created     []domain.APIKey
}

func (s *fakeAPIKeyStore) CreateAPIKey(_ context.Context, key domain.APIKey) (domain.APIKey, error) {
	if s.itemsByHash == nil {
		s.itemsByHash = map[string]domain.APIKey{}
	}
	s.itemsByHash[key.TokenHash] = key
	s.created = append(s.created, key)
	return key, nil
}

func (s *fakeAPIKeyStore) GetByTokenHash(_ context.Context, tokenHash string) (domain.APIKey, error) {
	key, ok := s.itemsByHash[tokenHash]
	if !ok || key.Revoked {
		return domain.APIKey{}, lib.NotFound("API_KEY_NOT_FOUND", "api key not found")
	}
	return key, nil
}

func (s *fakeAPIKeyStore) ListAPIKeys(_ context.Context) ([]domain.APIKey, error) {
	var items []domain.APIKey
	for _, item := range s.itemsByHash {
		items = append(items, item)
	}
	for _, item := range s.created {
		seen := false
		for _, existing := range items {
			if existing.ID == item.ID {
				seen = true
				break
			}
		}
		if !seen {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *fakeAPIKeyStore) RevokeAPIKey(_ context.Context, keyID string) (domain.APIKey, error) {
	for hash, item := range s.itemsByHash {
		if item.ID == keyID {
			item.Revoked = true
			s.itemsByHash[hash] = item
			return item, nil
		}
	}
	for i, item := range s.created {
		if item.ID == keyID {
			item.Revoked = true
			s.created[i] = item
			return item, nil
		}
	}
	return domain.APIKey{}, lib.NotFound("API_KEY_NOT_FOUND_BY_ID", "api key not found")
}

func TestCaptureCreatesProjectNoteAndArtifacts(t *testing.T) {
	projects := &fakeProjectStore{}
	notes := &fakeNoteStore{}
	artifacts := &fakeArtifactStore{}
	service := New(Dependencies{
		Projects:      projects,
		Notes:         notes,
		Artifacts:     artifacts,
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	result, err := service.Capture(context.Background(), CaptureInput{
		RepoPath:    "/tmp/model-manager",
		HandoffPath: "docs/handoff.md",
		DesignPath:  "docs/brainstorm.md",
		ExtraArtifacts: []CaptureArtifactInput{
			{Type: "code_path", SourcePath: "internal/services/capture.go"},
			{Type: "changed_files", SourcePath: "scripts/evals/fixtures/changed-files/api-first-boundary.txt"},
		},
		Note: "Need to confirm fallback behavior",
	})
	if err != nil {
		t.Fatalf("Capture returned error: %v", err)
	}
	if result.ProjectID == "" {
		t.Fatalf("expected project id")
	}
	if len(result.CreatedNoteIDs) != 1 {
		t.Fatalf("expected 1 note, got %d", len(result.CreatedNoteIDs))
	}
	if len(result.CreatedArtifactIDs) != 5 {
		t.Fatalf("expected 5 artifacts, got %d", len(result.CreatedArtifactIDs))
	}
	if len(artifacts.items) != 5 {
		t.Fatalf("expected 5 stored artifacts, got %d", len(artifacts.items))
	}
	if artifacts.items[3].Type != "code_path" || artifacts.items[3].TrustLevel != "trusted" {
		t.Fatalf("expected code_path extra artifact with default trusted level, got %#v", artifacts.items[3])
	}
}

func TestCaptureRejectsExtraArtifactWithoutRequiredFields(t *testing.T) {
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: lib.ProjectID("relay"), Name: "relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	_, err := service.Capture(context.Background(), CaptureInput{
		Project: "relay",
		ExtraArtifacts: []CaptureArtifactInput{
			{Type: "code_path"},
		},
	})
	if err == nil {
		t.Fatal("expected capture to reject extra artifact without source_path")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "MISSING_REQUIRED_FIELDS" {
		t.Fatalf("expected MISSING_REQUIRED_FIELDS, got %q", appErr.Code)
	}
	if len(appErr.MissingFields) != 1 || appErr.MissingFields[0] != "extra_artifacts[0].source_path" {
		t.Fatalf("unexpected missing fields: %#v", appErr.MissingFields)
	}
}

func TestCaptureUsesNoteAliasAndDefaultsSource(t *testing.T) {
	notes := &fakeNoteStore{}
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         notes,
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	result, err := service.Capture(context.Background(), CaptureInput{
		Note: "Alias body text",
	})
	if err != nil {
		t.Fatalf("Capture returned error: %v", err)
	}
	if result.ProjectID != "" {
		t.Fatalf("expected no project id, got %q", result.ProjectID)
	}
	if len(notes.items) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes.items))
	}
	if notes.items[0].Body != "Alias body text" {
		t.Fatalf("expected note body to come from note alias, got %q", notes.items[0].Body)
	}
	if notes.items[0].Source != "manual" {
		t.Fatalf("expected default source manual, got %q", notes.items[0].Source)
	}
}

func TestCaptureUsesBoundProjectForNoteOnlyProjectScopedKey(t *testing.T) {
	relayID := lib.ProjectID("relay")
	notes := &fakeNoteStore{}
	artifacts := &fakeArtifactStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay", RootPath: "/tmp/relay"},
			},
		},
		Notes:         notes,
		Artifacts:     artifacts,
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	result, err := service.Capture(ctx, CaptureInput{
		Note: "Bound project note only",
	})
	if err != nil {
		t.Fatalf("Capture returned error: %v", err)
	}
	if result.ProjectID != relayID {
		t.Fatalf("expected bound project id %q, got %q", relayID, result.ProjectID)
	}
	if len(notes.items) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes.items))
	}
	if notes.items[0].ProjectID != relayID {
		t.Fatalf("expected note to be stored against bound project, got %#v", notes.items[0])
	}
	if len(artifacts.items) != 0 {
		t.Fatalf("expected no artifacts without repo_path or document paths, got %#v", artifacts.items)
	}
}

func TestCaptureRejectsOtherProjectForProjectScopedKey(t *testing.T) {
	relayID := lib.ProjectID("relay")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay"},
				"other": {ID: lib.ProjectID("other"), Name: "other"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	_, err := service.Capture(ctx, CaptureInput{
		Project: "other",
		Source:  "chat",
		Body:    "hello",
	})
	if err == nil {
		t.Fatal("expected project-scoped key to reject capture into a different project")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestCaptureUsesBoundProjectForMatchingRepoPathDerivedName(t *testing.T) {
	relayID := lib.ProjectID("relay")
	notes := &fakeNoteStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay", RootPath: "/tmp/relay"},
			},
		},
		Notes:         notes,
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	result, err := service.Capture(ctx, CaptureInput{
		RepoPath: "/tmp/relay",
		Source:   "chat",
		Body:     "hello",
	})
	if err != nil {
		t.Fatalf("Capture returned error: %v", err)
	}
	if result.ProjectID != relayID {
		t.Fatalf("expected bound project id %q, got %q", relayID, result.ProjectID)
	}
	if len(notes.items) != 1 || notes.items[0].ProjectID != relayID {
		t.Fatalf("expected note to be stored against bound project, got %#v", notes.items)
	}
}

func TestCaptureRejectsRepoPathAliasForProjectScopedKey(t *testing.T) {
	relayID := lib.ProjectID("relay")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay", RootPath: "/tmp/relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	_, err := service.Capture(ctx, CaptureInput{
		RepoPath: "/tmp/custom-alias",
		Source:   "chat",
		Body:     "hello",
	})
	if err == nil {
		t.Fatal("expected repo_path alias to be rejected for project-scoped capture")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestCaptureRejectsRepoPathWithoutBoundRootPathForProjectScopedKey(t *testing.T) {
	relayID := lib.ProjectID("relay")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	_, err := service.Capture(ctx, CaptureInput{
		RepoPath: "/tmp/relay",
		Source:   "chat",
		Body:     "hello",
	})
	if err == nil {
		t.Fatal("expected repo_path to be rejected when the bound project has no stored root path")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestPromoteDecision(t *testing.T) {
	projects := &fakeProjectStore{
		projects: map[string]domain.Project{
			"relay": {ID: lib.ProjectID("relay"), Name: "relay"},
		},
	}
	decisions := &fakeDecisionStore{}
	service := New(Dependencies{
		Projects:      projects,
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     decisions,
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	result, err := service.Promote(context.Background(), PromoteInput{
		Project: "relay",
		Kind:    "decision",
		Summary: "v1 deploys PG-only",
		Reason:  "SaaS simplicity first",
	})
	if err != nil {
		t.Fatalf("Promote returned error: %v", err)
	}
	if result.Kind != "decision" {
		t.Fatalf("expected decision kind, got %s", result.Kind)
	}
	if len(decisions.items) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions.items))
	}
}

func TestPromoteRejectsOtherProjectForProjectScopedKey(t *testing.T) {
	relayID := lib.ProjectID("relay")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay"},
				"other": {ID: lib.ProjectID("other"), Name: "other"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	_, err := service.Promote(ctx, PromoteInput{
		Project: "other",
		Kind:    "decision",
		Summary: "ship it",
		Reason:  "because",
	})
	if err == nil {
		t.Fatal("expected project-scoped key to reject promote into a different project")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestBuildPacket(t *testing.T) {
	projectID := lib.ProjectID("relay")
	projects := &fakeProjectStore{
		projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		},
	}
	notes := &fakeNoteStore{
		items: []domain.Note{
			{ID: "note_1", ProjectID: projectID, Source: "chat", Body: "offline matters"},
		},
	}
	artifacts := &fakeArtifactStore{
		items: []domain.Artifact{
			{ID: "art_1", ProjectID: projectID, Type: "handoff_md", SourcePath: "docs/handoff.md", TrustLevel: "trusted"},
		},
	}
	decisions := &fakeDecisionStore{
		items: []domain.Decision{
			{ID: "dec_1", ProjectID: projectID, Summary: "PG-only deploy", Why: "simple SaaS path"},
		},
	}
	questions := &fakeOpenQuestionStore{
		items: []domain.OpenQuestion{
			{ID: "q_1", ProjectID: projectID, Summary: "offline local store timing"},
		},
	}
	packets := &fakePacketStore{}

	service := New(Dependencies{
		Projects:      projects,
		Notes:         notes,
		Artifacts:     artifacts,
		Decisions:     decisions,
		OpenQuestions: questions,
		Packets:       packets,
		APIKeys:       &fakeAPIKeyStore{},
	})

	result, err := service.BuildPacket(context.Background(), PacketBuildInput{
		Project: "relay",
		Type:    "resume",
		Target:  "codex",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	if result.PacketID == "" {
		t.Fatalf("expected packet id")
	}
	if result.Target != "codex" {
		t.Fatalf("expected codex target, got %s", result.Target)
	}
	if result.Body == "" {
		t.Fatalf("expected body")
	}
	if len(result.SupportingNotes) != 1 || !strings.Contains(result.SupportingNotes[0].Excerpt, "offline matters") {
		t.Fatalf("expected supporting note excerpt, got %#v", result.SupportingNotes)
	}
	if len(result.SupportingDecisions) != 1 || result.SupportingDecisions[0].Summary != "PG-only deploy" {
		t.Fatalf("expected supporting decision, got %#v", result.SupportingDecisions)
	}
	if len(result.SupportingQuestions) != 1 || result.SupportingQuestions[0].Summary != "offline local store timing" {
		t.Fatalf("expected supporting question, got %#v", result.SupportingQuestions)
	}
	if len(result.SupportingArtifacts) != 1 || result.SupportingArtifacts[0].SourcePath != "docs/handoff.md" {
		t.Fatalf("expected supporting artifact, got %#v", result.SupportingArtifacts)
	}
	if len(result.WhyIncluded) != 4 {
		t.Fatalf("expected why_included reasons, got %#v", result.WhyIncluded)
	}
	if !strings.Contains(result.RenderedBody, "Recent notes:") || !strings.Contains(result.RenderedBody, "Durable decisions:") {
		t.Fatalf("expected enriched rendered body, got %q", result.RenderedBody)
	}
	if packets.latest.ID == "" {
		t.Fatalf("expected packet to be persisted")
	}
}

func TestBuildPacketRanksArtifactsByTaskSummary(t *testing.T) {
	projectID := lib.ProjectID("relay")
	projects := &fakeProjectStore{
		projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		},
	}
	artifacts := &fakeArtifactStore{
		items: []domain.Artifact{
			{ID: "art_api", ProjectID: projectID, Type: "design_doc", SourcePath: "docs/api.md", TrustLevel: "trusted"},
			{ID: "art_packets", ProjectID: projectID, Type: "code_path", SourcePath: "internal/services/packets.go", TrustLevel: "trusted"},
			{ID: "art_repo", ProjectID: projectID, Type: "repo_path", SourcePath: "cmd/relay-worker/main.go", TrustLevel: "trusted"},
			{ID: "art_handoff", ProjectID: projectID, Type: "handoff_md", SourcePath: "docs/evals/v1-canonical-handoff.md", TrustLevel: "trusted"},
			{ID: "art_design", ProjectID: projectID, Type: "design_doc", SourcePath: "docs/research/context-graph-and-semantic-retrieval.md", TrustLevel: "trusted"},
			{ID: "art_capture", ProjectID: projectID, Type: "code_path", SourcePath: "internal/services/capture.go", TrustLevel: "trusted"},
			{ID: "art_report", ProjectID: projectID, Type: "code_path", SourcePath: "scripts/evals/v1_usage_validation_report.py", TrustLevel: "trusted"},
			{ID: "art_mcp", ProjectID: projectID, Type: "design_doc", SourcePath: "docs/mcp.md", TrustLevel: "trusted"},
			{ID: "art_worker", ProjectID: projectID, Type: "code_path", SourcePath: "cmd/relay-worker/main.go", TrustLevel: "trusted"},
			{ID: "art_curator", ProjectID: projectID, Type: "code_path", SourcePath: "internal/services/curator.go", TrustLevel: "trusted"},
		},
	}

	service := New(Dependencies{
		Projects:      projects,
		Notes:         &fakeNoteStore{},
		Artifacts:     artifacts,
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	result, err := service.BuildPacket(context.Background(), PacketBuildInput{
		Project:     "relay",
		Type:        "resume",
		Target:      "codex",
		TaskSummary: "Continue API packet work by checking docs/api.md and internal/services/packets.go before wrapper changes.",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	if len(result.SupportingArtifacts) != 8 {
		t.Fatalf("expected 8 supporting artifacts, got %d", len(result.SupportingArtifacts))
	}
	if result.SupportingArtifacts[0].SourcePath != "internal/services/packets.go" {
		t.Fatalf("expected packets.go to rank first, got %#v", result.SupportingArtifacts)
	}
	if !containsArtifactPath(result.SupportingArtifacts, "docs/api.md") {
		t.Fatalf("expected docs/api.md to be retained by task ranking, got %#v", result.SupportingArtifacts)
	}
	if containsArtifactPath(result.SupportingArtifacts, "docs/evals/v1-canonical-handoff.md") {
		t.Fatalf("expected a lower-signal artifact to be displaced by task-ranked evidence, got %#v", result.SupportingArtifacts)
	}
	if !strings.Contains(result.SupportingArtifacts[0].WhyIncluded, "task summary") {
		t.Fatalf("expected why_included to explain ranking, got %#v", result.SupportingArtifacts[0])
	}
}

func TestScoreArtifactForTaskIgnoresGenericRepoRootName(t *testing.T) {
	taskSummary := "Resume Relay contract work by checking API-visible behavior."
	score, whyIncluded := scoreArtifactForTask(
		domain.Artifact{Type: "git_commits", SourcePath: "/Users/hoon-ch/repos/relay", TrustLevel: "trusted"},
		tokenizeRankingText(taskSummary),
		strings.ToLower(taskSummary),
	)
	if score != 0 {
		t.Fatalf("expected generic repo-root artifact to avoid task-match scoring, got score=%d why=%q", score, whyIncluded)
	}
	if whyIncluded != "recent trusted artifact retained as fallback evidence" {
		t.Fatalf("expected fallback why_included, got %q", whyIncluded)
	}
}

func containsArtifactPath(items []PacketArtifact, sourcePath string) bool {
	for _, item := range items {
		if item.SourcePath == sourcePath {
			return true
		}
	}
	return false
}

func TestBuildPacketRejectsOtherProjectForProjectScopedKey(t *testing.T) {
	relayID := lib.ProjectID("relay")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay"},
				"other": {ID: lib.ProjectID("other"), Name: "other"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	_, err := service.BuildPacket(ctx, PacketBuildInput{
		Project: "other",
		Type:    "resume",
		Target:  "codex",
	})
	if err == nil {
		t.Fatal("expected project-scoped key to reject packet build for a different project")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestProjectScopedAccessRejectsMissingAndOtherProjectsWithoutLeak(t *testing.T) {
	relayID := lib.ProjectID("relay")
	otherID := lib.ProjectID("other")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay"},
				"other": {ID: otherID, Name: "other"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "show-other",
			run: func() error {
				_, err := service.Show(ctx, ShowInput{ProjectID: otherID})
				return err
			},
		},
		{
			name: "show-missing",
			run: func() error {
				_, err := service.Show(ctx, ShowInput{ProjectID: lib.ProjectID("missing")})
				return err
			},
		},
		{
			name: "promote-other",
			run: func() error {
				_, err := service.Promote(ctx, PromoteInput{Project: "other", Kind: "decision", Summary: "ship it", Reason: "because"})
				return err
			},
		},
		{
			name: "promote-missing",
			run: func() error {
				_, err := service.Promote(ctx, PromoteInput{Project: "missing", Kind: "decision", Summary: "ship it", Reason: "because"})
				return err
			},
		},
		{
			name: "packet-other",
			run: func() error {
				_, err := service.BuildPacket(ctx, PacketBuildInput{Project: "other", Type: "resume", Target: "codex"})
				return err
			},
		},
		{
			name: "packet-missing",
			run: func() error {
				_, err := service.BuildPacket(ctx, PacketBuildInput{Project: "missing", Type: "resume", Target: "codex"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatal("expected error")
			}
			appErr, ok := err.(lib.AppError)
			if !ok {
				t.Fatalf("expected AppError, got %T", err)
			}
			if appErr.Code != "FORBIDDEN" {
				t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
			}
		})
	}
}

func TestShowByProjectID(t *testing.T) {
	projectID := lib.ProjectID("relay")
	projects := &fakeProjectStore{
		projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		},
	}
	notes := &fakeNoteStore{
		items: []domain.Note{
			{ID: "note_1", ProjectID: projectID, Source: "chat", Body: "hello"},
		},
	}

	service := New(Dependencies{
		Projects:      projects,
		Notes:         notes,
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	result, err := service.Show(context.Background(), ShowInput{ProjectID: projectID})
	if err != nil {
		t.Fatalf("Show returned error: %v", err)
	}
	if result.ProjectID != projectID {
		t.Fatalf("expected project id %s, got %s", projectID, result.ProjectID)
	}
	if result.NoteCount != 1 {
		t.Fatalf("expected 1 note, got %d", result.NoteCount)
	}
}

func TestShowAllowsBoundProjectForProjectScopedKey(t *testing.T) {
	projectID := lib.ProjectID("relay")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: projectID, Name: "relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: projectID,
	})

	if _, err := service.Show(ctx, ShowInput{ProjectID: projectID}); err != nil {
		t.Fatalf("Show returned error: %v", err)
	}
}

func TestShowRejectsOtherProjectForProjectScopedKey(t *testing.T) {
	relayID := lib.ProjectID("relay")
	otherID := lib.ProjectID("other")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay"},
				"other": {ID: otherID, Name: "other"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID:     "key_1",
		Scope:     APIKeyScopeProject,
		ProjectID: relayID,
	})

	_, err := service.Show(ctx, ShowInput{ProjectID: otherID})
	if err == nil {
		t.Fatal("expected project-scoped key to reject a different project")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestShowRejectsMalformedProjectScopedAuth(t *testing.T) {
	projectID := lib.ProjectID("relay")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: projectID, Name: "relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID: "key_1",
		Scope: APIKeyScopeProject,
	})

	_, err := service.Show(ctx, ShowInput{ProjectID: projectID})
	if err == nil {
		t.Fatal("expected malformed project-scoped auth to be rejected")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestShowAllowsGlobalKeyAcrossProjects(t *testing.T) {
	relayID := lib.ProjectID("relay")
	otherID := lib.ProjectID("other")
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: relayID, Name: "relay"},
				"other": {ID: otherID, Name: "other"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID: "key_1",
		Scope: APIKeyScopeGlobal,
	})

	if _, err := service.Show(ctx, ShowInput{ProjectID: otherID}); err != nil {
		t.Fatalf("Show returned error: %v", err)
	}
}

func TestIssueAPIKey(t *testing.T) {
	keys := &fakeAPIKeyStore{}
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       keys,
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		IsAdmin: true,
		Scope:   APIKeyScopeGlobal,
	})

	result, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{Name: "agent"})
	if err != nil {
		t.Fatalf("IssueAPIKey returned error: %v", err)
	}
	if result.KeyID == "" || result.Token == "" {
		t.Fatalf("expected issued key id and token, got %#v", result)
	}
	if len(keys.created) != 1 {
		t.Fatalf("expected one created key, got %d", len(keys.created))
	}
	if keys.created[0].TokenHash != lib.TokenHash(result.Token) {
		t.Fatalf("expected stored hash to match returned token")
	}
}

func TestIssueAPIKeyRejectsInvalidScope(t *testing.T) {
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		IsAdmin: true,
		Scope:   APIKeyScopeGlobal,
	})

	_, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{Name: "agent", Scope: "invalid"})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "INVALID_API_KEY_SCOPE" {
		t.Fatalf("expected INVALID_API_KEY_SCOPE, got %q", appErr.Code)
	}
}

func TestIssueAPIKeyRejectsProjectBindingWithoutProjectScope(t *testing.T) {
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		IsAdmin: true,
		Scope:   APIKeyScopeGlobal,
	})

	_, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{
		Name:    "agent",
		Project: "relay",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "INVALID_API_KEY_SCOPE" {
		t.Fatalf("expected INVALID_API_KEY_SCOPE, got %q", appErr.Code)
	}
}

func TestIssueAPIKeyPersistsProjectScopeBinding(t *testing.T) {
	projectID := lib.ProjectID("relay")
	keys := &fakeAPIKeyStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: projectID, Name: "relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       keys,
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		IsAdmin: true,
		Scope:   APIKeyScopeGlobal,
	})

	result, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{
		Name:    "agent",
		Scope:   APIKeyScopeProject,
		Project: "relay",
	})
	if err != nil {
		t.Fatalf("IssueAPIKey returned error: %v", err)
	}
	if result.Scope != APIKeyScopeProject {
		t.Fatalf("expected project scope, got %q", result.Scope)
	}
	if result.ProjectID != projectID {
		t.Fatalf("expected project id %q, got %q", projectID, result.ProjectID)
	}
	if len(keys.created) != 1 {
		t.Fatalf("expected one created key, got %d", len(keys.created))
	}
	if keys.created[0].Scope != APIKeyScopeProject {
		t.Fatalf("expected stored project scope, got %q", keys.created[0].Scope)
	}
	if keys.created[0].ProjectID != projectID {
		t.Fatalf("expected stored project id %q, got %q", projectID, keys.created[0].ProjectID)
	}
}

func TestIssueAPIKeyRejectsProjectMismatch(t *testing.T) {
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: lib.ProjectID("proj_relay"), Name: "relay"},
				"other": {ID: lib.ProjectID("proj_other"), Name: "other"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		IsAdmin: true,
		Scope:   APIKeyScopeGlobal,
	})

	_, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{
		Name:      "agent",
		Scope:     APIKeyScopeProject,
		Project:   "relay",
		ProjectID: lib.ProjectID("proj_other"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "PROJECT_MISMATCH" {
		t.Fatalf("expected PROJECT_MISMATCH, got %q", appErr.Code)
	}
}

func TestListAPIKeys(t *testing.T) {
	keys := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			"hash": {ID: "key_1", Name: "agent", TokenHash: "hash", TokenPrefix: "relay_live_abc"},
		},
	}
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       keys,
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		IsAdmin: true,
		Scope:   APIKeyScopeGlobal,
	})

	result, err := service.ListAPIKeys(ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(result.Items))
	}
}

func TestRevokeAPIKey(t *testing.T) {
	keys := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			"hash": {ID: "key_1", Name: "agent", TokenHash: "hash", TokenPrefix: "relay_live_abc"},
		},
	}
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       keys,
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		IsAdmin: true,
		Scope:   APIKeyScopeGlobal,
	})

	result, err := service.RevokeAPIKey(ctx, RevokeAPIKeyInput{KeyID: "key_1"})
	if err != nil {
		t.Fatalf("RevokeAPIKey returned error: %v", err)
	}
	if !result.Revoked {
		t.Fatalf("expected revoked result")
	}
}

func TestAdminMethodsRejectNonAdminAuthContext(t *testing.T) {
	keys := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			"hash": {ID: "key_1", Name: "agent", TokenHash: "hash", TokenPrefix: "relay_live_abc"},
		},
	}
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       keys,
	})

	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
		KeyID: "key_1",
		Scope: APIKeyScopeGlobal,
	})

	tests := []struct {
		name string
		run  func(context.Context) error
	}{
		{
			name: "issue",
			run: func(ctx context.Context) error {
				_, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{Name: "agent"})
				return err
			},
		},
		{
			name: "list",
			run: func(ctx context.Context) error {
				_, err := service.ListAPIKeys(ctx)
				return err
			},
		},
		{
			name: "revoke",
			run: func(ctx context.Context) error {
				_, err := service.RevokeAPIKey(ctx, RevokeAPIKeyInput{KeyID: "key_1"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(ctx)
			if err == nil {
				t.Fatal("expected non-admin auth context to be rejected")
			}
			appErr, ok := err.(lib.AppError)
			if !ok {
				t.Fatalf("expected AppError, got %T", err)
			}
			if appErr.Code != "FORBIDDEN" {
				t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
			}
		})
	}
}

func TestUserControlledStringFieldsRejectOverlongValues(t *testing.T) {
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: lib.ProjectID("relay"), Name: "relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	longText := strings.Repeat("a", maxCaptureTextLength+1)
	longKey := strings.Repeat("k", maxAPIKeyIDLength+1)
	longName := strings.Repeat("n", maxAPIKeyNameLength+1)
	longTarget := strings.Repeat("t", maxPacketTargetLength+1)

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "capture body",
			run: func() error {
				_, err := service.Capture(context.Background(), CaptureInput{
					Project: "relay",
					Source:  "chat",
					Body:    longText,
				})
				return err
			},
		},
		{
			name: "promote summary",
			run: func() error {
				_, err := service.Promote(context.Background(), PromoteInput{
					Project: "relay",
					Kind:    "decision",
					Summary: longText,
					Reason:  "because",
				})
				return err
			},
		},
		{
			name: "packet target",
			run: func() error {
				_, err := service.BuildPacket(context.Background(), PacketBuildInput{
					Project: "relay",
					Type:    "resume",
					Target:  longTarget,
				})
				return err
			},
		},
		{
			name: "api key name",
			run: func() error {
				ctx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})
				_, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{Name: longName})
				return err
			},
		},
		{
			name: "revoke key id",
			run: func() error {
				ctx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})
				_, err := service.RevokeAPIKey(ctx, RevokeAPIKeyInput{KeyID: longKey})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatal("expected overlong input to be rejected")
			}
			appErr, ok := err.(lib.AppError)
			if !ok {
				t.Fatalf("expected AppError, got %T", err)
			}
			if appErr.Code != "FIELD_TOO_LONG" {
				t.Fatalf("expected FIELD_TOO_LONG, got %q", appErr.Code)
			}
			if appErr.Message == "" {
				t.Fatal("expected validation message")
			}
		})
	}
}

func TestValidateStringFieldLengthCountsUTF8Characters(t *testing.T) {
	shortUTF8 := strings.Repeat("界", 3)
	if err := validateStringFieldLength("name", shortUTF8, 4); err != nil {
		t.Fatalf("expected rune-based validation to accept %q, got %v", shortUTF8, err)
	}

	longUTF8 := strings.Repeat("界", 5)
	err := validateStringFieldLength("name", longUTF8, 4)
	if err == nil {
		t.Fatal("expected overlong UTF-8 input to be rejected")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FIELD_TOO_LONG" {
		t.Fatalf("expected FIELD_TOO_LONG, got %q", appErr.Code)
	}
	if appErr.Message != "name exceeds maximum length of 4 characters" {
		t.Fatalf("expected character-based message, got %q", appErr.Message)
	}
}

func TestCaptureRejectsOverlongDerivedProjectName(t *testing.T) {
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	_, err := service.Capture(context.Background(), CaptureInput{
		RepoPath: strings.Repeat("a", maxCaptureProjectLength+1),
		Source:   "chat",
		Body:     "hello",
	})
	if err == nil {
		t.Fatal("expected derived project name to be validated")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "FIELD_TOO_LONG" {
		t.Fatalf("expected FIELD_TOO_LONG, got %q", appErr.Code)
	}
}

func TestPromoteRejectsTooManyOrTooLongSourceIDs(t *testing.T) {
	service := New(Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: lib.ProjectID("relay"), Name: "relay"},
			},
		},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       &fakeAPIKeyStore{},
	})

	tooManyNoteIDs := make([]string, maxPromoteSourceIDs+1)
	for i := range tooManyNoteIDs {
		tooManyNoteIDs[i] = "note_1"
	}

	tests := []struct {
		name  string
		input PromoteInput
		code  string
	}{
		{
			name: "note count",
			input: PromoteInput{
				Project:           "relay",
				Kind:              "decision",
				Summary:           "ship it",
				Reason:            "because",
				SourceNoteIDs:     tooManyNoteIDs,
				SourceArtifactIDs: nil,
			},
			code: "FIELD_TOO_MANY_ITEMS",
		},
		{
			name: "artifact id length",
			input: PromoteInput{
				Project:           "relay",
				Kind:              "decision",
				Summary:           "ship it",
				Reason:            "because",
				SourceNoteIDs:     nil,
				SourceArtifactIDs: []string{strings.Repeat("a", maxPromoteSourceIDLength+1)},
			},
			code: "FIELD_TOO_LONG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Promote(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected promote input to be rejected")
			}
			appErr, ok := err.(lib.AppError)
			if !ok {
				t.Fatalf("expected AppError, got %T", err)
			}
			if appErr.Code != tt.code {
				t.Fatalf("expected %s, got %q", tt.code, appErr.Code)
			}
		})
	}
}

func TestAdminMethodsRejectMissingAuthContext(t *testing.T) {
	keys := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			"hash": {ID: "key_1", Name: "agent", TokenHash: "hash", TokenPrefix: "relay_live_abc"},
		},
	}
	service := New(Dependencies{
		Projects:      &fakeProjectStore{},
		Notes:         &fakeNoteStore{},
		Artifacts:     &fakeArtifactStore{},
		Decisions:     &fakeDecisionStore{},
		OpenQuestions: &fakeOpenQuestionStore{},
		Packets:       &fakePacketStore{},
		APIKeys:       keys,
	})

	tests := []struct {
		name string
		run  func(context.Context) error
	}{
		{
			name: "issue",
			run: func(ctx context.Context) error {
				_, err := service.IssueAPIKey(ctx, IssueAPIKeyInput{Name: "agent"})
				return err
			},
		},
		{
			name: "list",
			run: func(ctx context.Context) error {
				_, err := service.ListAPIKeys(ctx)
				return err
			},
		},
		{
			name: "revoke",
			run: func(ctx context.Context) error {
				_, err := service.RevokeAPIKey(ctx, RevokeAPIKeyInput{KeyID: "key_1"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(context.Background())
			if err == nil {
				t.Fatal("expected missing auth context to be rejected")
			}
			appErr, ok := err.(lib.AppError)
			if !ok {
				t.Fatalf("expected AppError, got %T", err)
			}
			if appErr.Code != "FORBIDDEN" {
				t.Fatalf("expected FORBIDDEN, got %q", appErr.Code)
			}
		})
	}
}
