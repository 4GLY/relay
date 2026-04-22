package services

import (
	"context"
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
		Note:        "Need to confirm fallback behavior",
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
	if len(result.CreatedArtifactIDs) != 3 {
		t.Fatalf("expected 3 artifacts, got %d", len(result.CreatedArtifactIDs))
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
	if packets.latest.ID == "" {
		t.Fatalf("expected packet to be persisted")
	}
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
