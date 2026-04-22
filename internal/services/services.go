package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/storage/repositories"
)

type Dependencies struct {
	Projects      repositories.ProjectStore
	Notes         repositories.NoteStore
	Artifacts     repositories.ArtifactStore
	Decisions     repositories.DecisionStore
	OpenQuestions repositories.OpenQuestionStore
	Packets       repositories.PacketStore
	APIKeys       repositories.APIKeyStore
}

type Service struct {
	deps Dependencies
}

func New(deps Dependencies) Service {
	return Service{deps: deps}
}

func (s Service) Capture(ctx context.Context, input CaptureInput) (CaptureResult, error) {
	projectName := input.Project
	if projectName == "" && input.RepoPath != "" {
		projectName = filepath.Base(input.RepoPath)
	}

	if err := validateCaptureInput(input); err != nil {
		return CaptureResult{}, err
	}
	if projectName != "" {
		if err := validateStringFieldLength("project", projectName, maxCaptureProjectLength); err != nil {
			return CaptureResult{}, err
		}
	}

	projectID := ""
	if projectName != "" {
		project, err := s.resolveCaptureProject(ctx, projectName, input.RepoPath)
		if err != nil {
			return CaptureResult{}, err
		}
		projectID = project.ID
	} else if auth, ok := AuthInfoFromContext(ctx); ok && NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeProject {
		project, err := s.resolveCaptureProject(ctx, "", input.RepoPath)
		if err != nil {
			return CaptureResult{}, err
		}
		projectID = project.ID
	}

	result := CaptureResult{ProjectID: projectID}

	if input.Body != "" || input.Note != "" {
		body := input.Body
		source := input.Source
		if body == "" {
			body = input.Note
		}
		if source == "" {
			source = "manual"
		}

		noteID := noteIDFor(input.IdempotencyKey, body)
		note, err := s.deps.Notes.CreateNote(ctx, domain.Note{
			ID:        noteID,
			ProjectID: projectID,
			Source:    source,
			Body:      body,
		})
		if err != nil {
			return CaptureResult{}, err
		}
		result.CreatedNoteIDs = append(result.CreatedNoteIDs, note.ID)
	}

	artifactSpecs := []struct {
		kind     string
		path     string
		trust    string
		idSuffix string
	}{
		{kind: "git_commits", path: input.RepoPath, trust: "trusted", idSuffix: "repo"},
		{kind: "handoff_md", path: input.HandoffPath, trust: "trusted", idSuffix: "handoff"},
		{kind: "design_doc", path: input.DesignPath, trust: "trusted", idSuffix: "design"},
	}

	for _, spec := range artifactSpecs {
		if spec.path == "" || projectID == "" {
			continue
		}
		artifactID := artifactIDFor(input.IdempotencyKey, spec.idSuffix, spec.path)
		artifact, err := s.deps.Artifacts.CreateArtifact(ctx, domain.Artifact{
			ID:         artifactID,
			ProjectID:  projectID,
			Type:       spec.kind,
			SourcePath: spec.path,
			TrustLevel: spec.trust,
		})
		if err != nil {
			return CaptureResult{}, err
		}
		result.CreatedArtifactIDs = append(result.CreatedArtifactIDs, artifact.ID)
	}

	return result, nil
}

func (s Service) Promote(ctx context.Context, input PromoteInput) (PromoteResult, error) {
	if err := validatePromoteInput(input); err != nil {
		return PromoteResult{}, err
	}

	if input.Project == "" {
		return PromoteResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}
	if input.Kind == "" {
		return PromoteResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "kind")
	}
	if input.Summary == "" {
		return PromoteResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "summary")
	}

	project, err := s.resolveProject(ctx, input.Project, "")
	if err != nil {
		return PromoteResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return PromoteResult{}, err
	}

	switch input.Kind {
	case "decision":
		if input.Reason == "" {
			return PromoteResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "reason")
		}
		id := promotedID("dec", input.IdempotencyKey, input.Summary)
		decision, err := s.deps.Decisions.CreateDecision(ctx, domain.Decision{
			ID:                id,
			ProjectID:         project.ID,
			Summary:           input.Summary,
			Why:               input.Reason,
			SourceNoteIDs:     input.SourceNoteIDs,
			SourceArtifactIDs: input.SourceArtifactIDs,
		})
		if err != nil {
			return PromoteResult{}, err
		}
		return PromoteResult{Kind: input.Kind, ObjectID: decision.ID, ProjectID: project.ID}, nil
	case "question":
		id := promotedID("q", input.IdempotencyKey, input.Summary)
		question, err := s.deps.OpenQuestions.CreateOpenQuestion(ctx, domain.OpenQuestion{
			ID:                id,
			ProjectID:         project.ID,
			Summary:           input.Summary,
			SourceNoteIDs:     input.SourceNoteIDs,
			SourceArtifactIDs: input.SourceArtifactIDs,
		})
		if err != nil {
			return PromoteResult{}, err
		}
		return PromoteResult{Kind: input.Kind, ObjectID: question.ID, ProjectID: project.ID}, nil
	default:
		return PromoteResult{}, lib.MissingFields("INVALID_KIND", "kind")
	}
}

func (s Service) Show(ctx context.Context, input ShowInput) (ShowResult, error) {
	if input.Project == "" && input.ProjectID == "" {
		return ShowResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}
	project, err := s.resolveProject(ctx, input.Project, input.ProjectID)
	if err != nil {
		return ShowResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return ShowResult{}, err
	}
	notes, err := s.deps.Notes.CountByProject(ctx, project.ID)
	if err != nil {
		return ShowResult{}, err
	}
	artifacts, err := s.deps.Artifacts.CountByProject(ctx, project.ID)
	if err != nil {
		return ShowResult{}, err
	}
	decisions, err := s.deps.Decisions.CountByProject(ctx, project.ID)
	if err != nil {
		return ShowResult{}, err
	}
	questions, err := s.deps.OpenQuestions.CountByProject(ctx, project.ID)
	if err != nil {
		return ShowResult{}, err
	}
	packet, err := s.deps.Packets.LatestByProject(ctx, project.ID)
	if err != nil {
		return ShowResult{}, err
	}
	return ShowResult{
		ProjectID:         project.ID,
		NoteCount:         notes,
		ArtifactCount:     artifacts,
		DecisionCount:     decisions,
		OpenQuestionCount: questions,
		LatestPacketID:    packet.ID,
	}, nil
}

func (s Service) BuildPacket(ctx context.Context, input PacketBuildInput) (PacketBuildResult, error) {
	if err := validatePacketBuildInput(input); err != nil {
		return PacketBuildResult{}, err
	}

	if input.Project == "" {
		return PacketBuildResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}

	packetType := input.Type
	if packetType == "" {
		packetType = "resume"
	}
	target := input.Target
	if target == "" {
		target = "generic"
	}

	project, err := s.resolveProject(ctx, input.Project, "")
	if err != nil {
		return PacketBuildResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return PacketBuildResult{}, err
	}

	notes, err := s.deps.Notes.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}
	artifacts, err := s.deps.Artifacts.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}
	decisions, err := s.deps.Decisions.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}
	questions, err := s.deps.OpenQuestions.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}

	packet := domain.Packet{
		ID:                lib.NewID("pkt"),
		ProjectID:         project.ID,
		Type:              packetType,
		Target:            target,
		Body:              buildResumeBody(project, notes, artifacts, decisions, questions),
		DecisionIDs:       collectDecisionIDs(decisions),
		OpenQuestionIDs:   collectQuestionIDs(questions),
		SourceArtifactIDs: collectArtifactIDs(artifacts),
	}

	created, err := s.deps.Packets.CreatePacket(ctx, packet)
	if err != nil {
		return PacketBuildResult{}, err
	}

	return PacketBuildResult{
		PacketID:          created.ID,
		ProjectID:         created.ProjectID,
		Type:              created.Type,
		Target:            created.Target,
		Body:              created.Body,
		DecisionIDs:       created.DecisionIDs,
		OpenQuestionIDs:   created.OpenQuestionIDs,
		SourceArtifactIDs: created.SourceArtifactIDs,
		MissingContext:    collectQuestionSummaries(questions),
	}, nil
}

func (s Service) IssueAPIKey(ctx context.Context, input IssueAPIKeyInput) (IssueAPIKeyResult, error) {
	if err := requireAdminAuth(ctx); err != nil {
		return IssueAPIKeyResult{}, err
	}
	if err := validateIssueAPIKeyInput(input); err != nil {
		return IssueAPIKeyResult{}, err
	}
	if input.Name == "" {
		return IssueAPIKeyResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "name")
	}
	if s.deps.APIKeys == nil {
		return IssueAPIKeyResult{}, lib.Misconfigured("api key store is required")
	}

	scope := NormalizeAPIKeyScope(input.Scope)
	if scope != APIKeyScopeGlobal && scope != APIKeyScopeProject {
		return IssueAPIKeyResult{}, lib.AppError{
			Code:      "INVALID_API_KEY_SCOPE",
			Message:   "api key scope must be global or project",
			Retryable: false,
		}
	}

	boundProjectID := ""
	if scope == APIKeyScopeProject {
		project, err := s.resolveProject(ctx, input.Project, input.ProjectID)
		if err != nil {
			return IssueAPIKeyResult{}, err
		}
		boundProjectID = project.ID
	}

	token, err := lib.NewSecretToken("relay_live")
	if err != nil {
		return IssueAPIKeyResult{}, err
	}

	key := domain.APIKey{
		ID:          lib.NewID("key"),
		Name:        input.Name,
		TokenHash:   lib.TokenHash(token),
		TokenPrefix: lib.TokenPrefix(token),
		Scope:       scope,
		ProjectID:   boundProjectID,
	}

	created, err := s.deps.APIKeys.CreateAPIKey(ctx, key)
	if err != nil {
		return IssueAPIKeyResult{}, err
	}

	return IssueAPIKeyResult{
		KeyID:       created.ID,
		Name:        created.Name,
		Token:       token,
		TokenPrefix: created.TokenPrefix,
		Scope:       NormalizeAPIKeyScope(created.Scope),
		ProjectID:   created.ProjectID,
	}, nil
}

func (s Service) ListAPIKeys(ctx context.Context) (ListAPIKeysResult, error) {
	if err := requireAdminAuth(ctx); err != nil {
		return ListAPIKeysResult{}, err
	}
	if s.deps.APIKeys == nil {
		return ListAPIKeysResult{}, lib.Misconfigured("api key store is required")
	}

	keys, err := s.deps.APIKeys.ListAPIKeys(ctx)
	if err != nil {
		return ListAPIKeysResult{}, err
	}

	items := make([]APIKeySummary, 0, len(keys))
	for _, key := range keys {
		items = append(items, APIKeySummary{
			KeyID:       key.ID,
			Name:        key.Name,
			TokenPrefix: key.TokenPrefix,
			Scope:       NormalizeAPIKeyScope(key.Scope),
			ProjectID:   key.ProjectID,
			Revoked:     key.Revoked,
		})
	}

	return ListAPIKeysResult{Items: items}, nil
}

func (s Service) RevokeAPIKey(ctx context.Context, input RevokeAPIKeyInput) (RevokeAPIKeyResult, error) {
	if err := requireAdminAuth(ctx); err != nil {
		return RevokeAPIKeyResult{}, err
	}
	if err := validateRevokeAPIKeyInput(input); err != nil {
		return RevokeAPIKeyResult{}, err
	}
	if input.KeyID == "" {
		return RevokeAPIKeyResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "key_id")
	}
	if s.deps.APIKeys == nil {
		return RevokeAPIKeyResult{}, lib.Misconfigured("api key store is required")
	}

	key, err := s.deps.APIKeys.RevokeAPIKey(ctx, input.KeyID)
	if err != nil {
		return RevokeAPIKeyResult{}, err
	}

	return RevokeAPIKeyResult{
		KeyID:       key.ID,
		Name:        key.Name,
		TokenPrefix: key.TokenPrefix,
		Scope:       NormalizeAPIKeyScope(key.Scope),
		ProjectID:   key.ProjectID,
		Revoked:     key.Revoked,
	}, nil
}

func noteIDFor(idempotencyKey string, body string) string {
	if idempotencyKey != "" {
		return lib.StableID("note", "capture:"+idempotencyKey+":note")
	}
	return lib.StableID("note", body)
}

func artifactIDFor(idempotencyKey string, suffix string, sourcePath string) string {
	if idempotencyKey != "" {
		return lib.StableID("art", "capture:"+idempotencyKey+":"+suffix)
	}
	return lib.StableID("art", suffix+":"+sourcePath)
}

func promotedID(prefix string, idempotencyKey string, summary string) string {
	if idempotencyKey != "" {
		return lib.StableID(prefix, "promote:"+idempotencyKey)
	}
	return lib.StableID(prefix, summary)
}

func (s Service) resolveProject(ctx context.Context, name string, id string) (domain.Project, error) {
	if auth, ok := AuthInfoFromContext(ctx); ok && NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeProject {
		return s.resolveBoundProject(ctx, auth, name, id)
	}
	if id != "" {
		project, err := s.deps.Projects.GetByID(ctx, id)
		if err != nil {
			return domain.Project{}, err
		}
		if name != "" && project.Name != name {
			return domain.Project{}, lib.AppError{
				Code:      "PROJECT_MISMATCH",
				Message:   "project and project_id do not match",
				Retryable: false,
			}
		}
		return project, nil
	}
	if name == "" {
		return domain.Project{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}
	return s.deps.Projects.GetByName(ctx, name)
}

func (s Service) resolveCaptureProject(ctx context.Context, name string, repoPath string) (domain.Project, error) {
	if auth, ok := AuthInfoFromContext(ctx); ok && NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeProject {
		if auth.ProjectID == "" {
			return domain.Project{}, lib.Forbidden("FORBIDDEN", "project-scoped api key is missing a project binding")
		}
		project, err := s.deps.Projects.GetByID(ctx, auth.ProjectID)
		if err != nil {
			return domain.Project{}, err
		}
		if name != "" && project.Name != name {
			return domain.Project{}, lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
		}
		return project, nil
	}

	project, err := s.deps.Projects.EnsureProject(ctx, domain.Project{
		ID:       lib.ProjectID(name),
		Name:     name,
		RootPath: repoPath,
		Status:   "active",
	})
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

func (s Service) resolveBoundProject(ctx context.Context, auth AuthInfo, name string, id string) (domain.Project, error) {
	if auth.ProjectID == "" {
		return domain.Project{}, lib.Forbidden("FORBIDDEN", "project-scoped api key is missing a project binding")
	}

	project, err := s.deps.Projects.GetByID(ctx, auth.ProjectID)
	if err != nil {
		return domain.Project{}, err
	}
	if id != "" && id != auth.ProjectID {
		return domain.Project{}, lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	if name != "" && project.Name != name {
		return domain.Project{}, lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	return project, nil
}

func (s Service) enforceProjectAccess(ctx context.Context, projectID string) error {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok {
		return nil
	}
	if NormalizeAPIKeyScope(auth.Scope) != APIKeyScopeProject {
		return nil
	}
	if auth.ProjectID == "" {
		return lib.Forbidden("FORBIDDEN", "project-scoped api key is missing a project binding")
	}
	if auth.ProjectID != projectID {
		return lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	return nil
}

func requireAdminAuth(ctx context.Context) error {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok {
		return lib.Forbidden("FORBIDDEN", "admin authorization is required")
	}
	if auth.IsAdmin {
		return nil
	}
	return lib.Forbidden("FORBIDDEN", "admin authorization is required")
}

func buildResumeBody(project domain.Project, notes []domain.Note, artifacts []domain.Artifact, decisions []domain.Decision, questions []domain.OpenQuestion) string {
	lines := []string{
		fmt.Sprintf("Project: %s", project.Name),
		fmt.Sprintf("Current goal: resume work on %s", project.Name),
		fmt.Sprintf("Current state: %d note(s), %d artifact(s), %d decision(s), %d open question(s) are stored.", len(notes), len(artifacts), len(decisions), len(questions)),
	}

	if len(decisions) > 0 {
		lines = append(lines, "Decisions:")
		for _, decision := range decisions {
			lines = append(lines, fmt.Sprintf("- %s: %s", decision.Summary, decision.Why))
		}
	}

	if len(questions) > 0 {
		lines = append(lines, "Open questions:")
		for _, question := range questions {
			lines = append(lines, fmt.Sprintf("- %s", question.Summary))
		}
	}

	if len(artifacts) > 0 {
		lines = append(lines, "Trusted artifacts:")
		for _, artifact := range artifacts {
			lines = append(lines, fmt.Sprintf("- %s (%s)", artifact.Type, artifact.SourcePath))
		}
	}

	lines = append(lines, "Next step: inspect the decisions and open questions, then continue the highest-confidence path.")
	return strings.Join(lines, "\n")
}

func collectDecisionIDs(items []domain.Decision) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func collectQuestionIDs(items []domain.OpenQuestion) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func collectArtifactIDs(items []domain.Artifact) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func collectQuestionSummaries(items []domain.OpenQuestion) []string {
	summaries := make([]string, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, item.Summary)
	}
	return summaries
}
