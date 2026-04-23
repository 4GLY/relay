package services

import (
	"context"
	"fmt"
	"strings"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

const packetSchemaVersionV1 = "relay.packet.v1"

func (s Service) BuildPacket(ctx context.Context, input PacketBuildInput) (PacketBuildResult, error) {
	if err := validatePacketBuildInput(input); err != nil {
		return PacketBuildResult{}, err
	}

	if input.Project == "" {
		return PacketBuildResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}

	packetType := contracts.NormalizePacketKind(input.Type)
	target := contracts.NormalizePacketTarget(input.Target)

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

	selectedDecisions := limitDecisions(decisions, 3)
	selectedQuestions := limitQuestions(questions, 3)
	selectedArtifacts := limitArtifacts(artifacts, 5)
	selectedNotes := limitNotes(notes, 3)
	styleCues, approvedHeuristicIDs, err := s.selectStyleCues(ctx, project.ID, input)
	if err != nil {
		return PacketBuildResult{}, err
	}
	taskSummary := input.TaskSummary
	if taskSummary == "" {
		taskSummary = fmt.Sprintf("resume work on %s", project.Name)
	}
	renderedBody := buildResumeBody(project, selectedNotes, selectedArtifacts, selectedDecisions, selectedQuestions)
	if len(styleCues) > 0 {
		renderedBody = appendStyleCues(renderedBody, styleCues)
	}

	packet := domain.Packet{
		ID:                lib.NewID("pkt"),
		ProjectID:         project.ID,
		Type:              packetType,
		Target:            target,
		Body:              renderedBody,
		DecisionIDs:       collectDecisionIDs(selectedDecisions),
		OpenQuestionIDs:   collectQuestionIDs(selectedQuestions),
		SourceArtifactIDs: collectArtifactIDs(selectedArtifacts),
	}

	created, err := s.deps.Packets.CreatePacket(ctx, packet)
	if err != nil {
		return PacketBuildResult{}, err
	}

	result := PacketBuildResult{
		PacketID:             created.ID,
		ProjectID:            created.ProjectID,
		SchemaVersion:        packetSchemaVersionV1,
		Type:                 created.Type,
		Target:               created.Target,
		TaskSummary:          taskSummary,
		Body:                 created.Body,
		RenderedBody:         created.Body,
		StyleCues:            styleCues,
		DecisionIDs:          created.DecisionIDs,
		OpenQuestionIDs:      created.OpenQuestionIDs,
		SourceArtifactIDs:    created.SourceArtifactIDs,
		ApprovedHeuristicIDs: approvedHeuristicIDs,
		MissingContext:       collectQuestionSummaries(selectedQuestions),
	}

	if input.PersistSnapshot {
		snapshot, err := s.createPacketSnapshot(ctx, project.ID, result, input.IdempotencyKey)
		if err != nil {
			return PacketBuildResult{}, err
		}
		result.SnapshotID = snapshot.ID
	}

	return result, nil
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

func (s Service) selectStyleCues(ctx context.Context, projectID string, input PacketBuildInput) ([]PacketStyleCue, []string, error) {
	if input.DisableStyleCues || s.deps.ApprovedHeuristics == nil {
		return nil, nil, nil
	}
	heuristics, err := s.deps.ApprovedHeuristics.ListApprovedHeuristicsByProject(ctx, projectID, input.Workflow, input.ArtifactType, 10)
	if err != nil {
		return nil, nil, err
	}
	heuristics = selectMostSpecificHeuristics(heuristics, 3)
	cues := make([]PacketStyleCue, 0, len(heuristics))
	ids := make([]string, 0, len(heuristics))
	for _, heuristic := range heuristics {
		ids = append(ids, heuristic.ID)
		scope := "this project"
		if heuristic.Workflow != "" {
			scope = "same workflow"
		}
		if heuristic.ArtifactType != "" {
			scope = "same workflow and artifact type"
		}
		cues = append(cues, PacketStyleCue{
			HeuristicID:   heuristic.ID,
			WhySelected:   "selected for " + scope,
			SourceSummary: fmt.Sprintf("%d source trace(s)", len(heuristic.SourceTraceIDs)),
		})
	}
	return cues, ids, nil
}

func selectMostSpecificHeuristics(items []domain.ApprovedHeuristic, limit int) []domain.ApprovedHeuristic {
	if len(items) <= 1 {
		return items
	}
	selected := make([]domain.ApprovedHeuristic, 0, minInt(len(items), limit))
	for _, item := range items {
		if len(selected) >= limit {
			break
		}
		selected = append(selected, item)
	}
	return selected
}

func appendStyleCues(body string, cues []PacketStyleCue) string {
	lines := []string{body, "Style cues:"}
	for _, cue := range cues {
		lines = append(lines, fmt.Sprintf("- %s: %s (%s)", cue.HeuristicID, cue.WhySelected, cue.SourceSummary))
	}
	return strings.Join(lines, "\n")
}

func (s Service) createPacketSnapshot(ctx context.Context, projectID string, result PacketBuildResult, idempotencyKey string) (domain.PacketSnapshot, error) {
	if s.deps.PacketSnapshots == nil {
		return domain.PacketSnapshot{}, lib.Misconfigured("packet snapshot store is required")
	}
	idempotencyPayload := result
	idempotencyPayload.PacketID = ""
	idempotencyPayload.SnapshotID = ""
	requestHash := normalizedRequestHash(idempotencyPayload)
	if lookup, err := s.lookupIdempotency(ctx, "packet_snapshot", projectID, idempotencyKey, requestHash); err != nil {
		return domain.PacketSnapshot{}, err
	} else if lookup.found {
		return s.deps.PacketSnapshots.GetPacketSnapshot(ctx, lookup.responseID)
	}

	snapshotID := lib.NewID("psnap")
	if idempotencyKey != "" {
		snapshotID = lib.StableID("psnap", projectID+":"+idempotencyKey)
	}
	snapshot, err := s.deps.PacketSnapshots.CreatePacketSnapshot(ctx, domain.PacketSnapshot{
		ID:                   snapshotID,
		ProjectID:            projectID,
		PacketKind:           result.Type,
		Target:               result.Target,
		SchemaVersion:        result.SchemaVersion,
		TaskSummary:          result.TaskSummary,
		RenderedBody:         result.RenderedBody,
		ApprovedHeuristicIDs: result.ApprovedHeuristicIDs,
		DecisionIDs:          result.DecisionIDs,
		OpenQuestionIDs:      result.OpenQuestionIDs,
		SourceArtifactIDs:    result.SourceArtifactIDs,
		MissingContext:       result.MissingContext,
	})
	if err != nil {
		return domain.PacketSnapshot{}, err
	}
	if err := s.recordIdempotency(ctx, "packet_snapshot", projectID, idempotencyKey, requestHash, "packet_snapshot", snapshot.ID); err != nil {
		return domain.PacketSnapshot{}, err
	}
	return snapshot, nil
}

func limitNotes(items []domain.Note, limit int) []domain.Note {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func limitArtifacts(items []domain.Artifact, limit int) []domain.Artifact {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func limitDecisions(items []domain.Decision, limit int) []domain.Decision {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func limitQuestions(items []domain.OpenQuestion, limit int) []domain.OpenQuestion {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
