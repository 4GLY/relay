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

	projectID := ""
	if projectName != "" {
		project, err := s.deps.Projects.EnsureProject(ctx, domain.Project{
			ID:       lib.ProjectID(projectName),
			Name:     projectName,
			RootPath: input.RepoPath,
			Status:   "active",
		})
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
	if id != "" {
		return s.deps.Projects.GetByID(ctx, id)
	}
	if name == "" {
		return domain.Project{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}
	return s.deps.Projects.GetByName(ctx, name)
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
