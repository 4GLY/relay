package services

import (
	"context"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
)

const (
	projectGraphEdgeIncludes    = "includes"
	projectGraphEdgeDerivedFrom = "derived_from"
)

type projectGraphLatestSnapshotStore interface {
	LatestAnyPacketSnapshotByProject(ctx context.Context, projectID string) (domain.PacketSnapshot, error)
}

func (s Service) ProjectGraph(ctx context.Context, input ProjectGraphInput) (ProjectGraphResult, error) {
	if input.Project == "" && input.ProjectID == "" {
		return ProjectGraphResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}
	project, err := s.resolveProject(ctx, input.Project, input.ProjectID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return ProjectGraphResult{}, err
	}

	notes, err := s.deps.Notes.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	artifacts, err := s.deps.Artifacts.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	decisions, err := s.deps.Decisions.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	questions, err := s.deps.OpenQuestions.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	traces, err := s.graphJudgmentTraces(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	proposals, err := s.graphHeuristicProposals(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	approvedHeuristics, err := s.graphApprovedHeuristics(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}
	latestSnapshot, err := s.graphLatestSnapshot(ctx, project.ID)
	if err != nil {
		return ProjectGraphResult{}, err
	}

	nodes := make([]ProjectGraphNode, 0, 1+len(notes)+len(artifacts)+len(decisions)+len(questions)+len(traces)+len(proposals)+len(approvedHeuristics)+1)
	edges := make([]ProjectGraphEdge, 0, len(notes)+len(artifacts)+len(decisions)*4+len(questions)*4+len(traces)+len(proposals)*3+len(approvedHeuristics)*3+6)
	knownNodeIDs := map[string]struct{}{
		project.ID: {},
	}

	nodes = append(nodes, ProjectGraphNode{
		ID:    project.ID,
		Kind:  "project",
		Title: project.Name,
	})

	for _, note := range notes {
		nodes = append(nodes, ProjectGraphNode{
			ID:     note.ID,
			Kind:   "note",
			Title:  summarizeText(note.Body, 120),
			Source: note.Source,
		})
		knownNodeIDs[note.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: note.ID})
	}

	for _, artifact := range artifacts {
		nodes = append(nodes, ProjectGraphNode{
			ID:         artifact.ID,
			Kind:       "artifact",
			Title:      artifact.Type,
			SourcePath: artifact.SourcePath,
			TrustLevel: artifact.TrustLevel,
		})
		knownNodeIDs[artifact.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: artifact.ID})
	}

	for _, decision := range decisions {
		nodes = append(nodes, ProjectGraphNode{
			ID:    decision.ID,
			Kind:  "decision",
			Title: decision.Summary,
		})
		knownNodeIDs[decision.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: decision.ID})
		edges = append(edges, projectGraphDerivedEdges(decision.ID, decision.SourceNoteIDs, decision.SourceArtifactIDs, knownNodeIDs)...)
	}

	for _, question := range questions {
		nodes = append(nodes, ProjectGraphNode{
			ID:    question.ID,
			Kind:  "open_question",
			Title: question.Summary,
		})
		knownNodeIDs[question.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: question.ID})
		edges = append(edges, projectGraphDerivedEdges(question.ID, question.SourceNoteIDs, question.SourceArtifactIDs, knownNodeIDs)...)
	}

	for _, trace := range traces {
		nodes = append(nodes, ProjectGraphNode{
			ID:           trace.ID,
			Kind:         "judgment_trace",
			Title:        firstNonEmpty(trace.Decision, trace.Rationale, trace.Workflow),
			Workflow:     trace.Workflow,
			ArtifactType: trace.ArtifactType,
			CreatedAt:    graphTime(trace.CreatedAt),
		})
		knownNodeIDs[trace.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: trace.ID})
	}

	for _, proposal := range proposals {
		nodes = append(nodes, ProjectGraphNode{
			ID:           proposal.ID,
			Kind:         "heuristic_proposal",
			Title:        firstNonEmpty(proposal.CanonicalText, proposal.HeuristicKey),
			Workflow:     proposal.Workflow,
			ArtifactType: proposal.ArtifactType,
			State:        proposal.State,
			CreatedAt:    graphTime(proposal.CreatedAt),
		})
		knownNodeIDs[proposal.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: proposal.ID})
		edges = append(edges, projectGraphDerivedEdges(proposal.ID, appendString(proposal.SourceTraceIDs, proposal.OriginTraceID), nil, knownNodeIDs)...)
	}

	for _, heuristic := range approvedHeuristics {
		nodes = append(nodes, ProjectGraphNode{
			ID:           heuristic.ID,
			Kind:         "approved_heuristic",
			Title:        firstNonEmpty(heuristic.CanonicalText, heuristic.HeuristicKey),
			Workflow:     heuristic.Workflow,
			ArtifactType: heuristic.ArtifactType,
			State:        heuristic.State,
			CreatedAt:    graphTime(heuristic.CreatedAt),
		})
		knownNodeIDs[heuristic.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: heuristic.ID})
		edges = append(edges, projectGraphDerivedEdges(heuristic.ID, appendString(heuristic.SourceTraceIDs, heuristic.OriginProposalID), nil, knownNodeIDs)...)
	}

	if latestSnapshot != nil {
		nodes = append(nodes, ProjectGraphNode{
			ID:             latestSnapshot.ID,
			Kind:           "packet_snapshot",
			Title:          firstNonEmpty(latestSnapshot.TaskSummary, latestSnapshot.Target, latestSnapshot.PacketKind),
			PacketKind:     latestSnapshot.PacketKind,
			Target:         latestSnapshot.Target,
			PublicReadable: latestSnapshot.PublicReadable,
			CreatedAt:      graphTime(latestSnapshot.CreatedAt),
		})
		knownNodeIDs[latestSnapshot.ID] = struct{}{}
		edges = append(edges, ProjectGraphEdge{Type: projectGraphEdgeIncludes, From: project.ID, To: latestSnapshot.ID})
		edges = append(edges, projectGraphDerivedEdges(latestSnapshot.ID, latestSnapshot.DecisionIDs, latestSnapshot.SourceArtifactIDs, knownNodeIDs)...)
		edges = append(edges, projectGraphDerivedEdges(latestSnapshot.ID, latestSnapshot.OpenQuestionIDs, nil, knownNodeIDs)...)
		edges = append(edges, projectGraphDerivedEdges(latestSnapshot.ID, latestSnapshot.ApprovedHeuristicIDs, nil, knownNodeIDs)...)
	}

	edges = append(edges, buildInferredSupportEdges(decisions, questions, notes, artifacts)...)

	return ProjectGraphResult{
		ProjectID: project.ID,
		Nodes:     nodes,
		Edges:     edges,
	}, nil
}

func (s Service) graphJudgmentTraces(ctx context.Context, projectID string) ([]domain.JudgmentTrace, error) {
	if s.deps.JudgmentTraces == nil {
		return nil, nil
	}
	return s.deps.JudgmentTraces.ListJudgmentTracesByProject(ctx, projectID, 100)
}

func (s Service) graphHeuristicProposals(ctx context.Context, projectID string) ([]domain.HeuristicProposal, error) {
	if s.deps.HeuristicProposals == nil {
		return nil, nil
	}
	return s.deps.HeuristicProposals.ListHeuristicProposalsByProject(ctx, projectID, "", "", 100)
}

func (s Service) graphApprovedHeuristics(ctx context.Context, projectID string) ([]domain.ApprovedHeuristic, error) {
	if s.deps.ApprovedHeuristics == nil {
		return nil, nil
	}
	return s.deps.ApprovedHeuristics.ListApprovedHeuristicsByProject(ctx, projectID, "", "", "", 100)
}

func (s Service) graphLatestSnapshot(ctx context.Context, projectID string) (*domain.PacketSnapshot, error) {
	store, ok := s.deps.PacketSnapshots.(projectGraphLatestSnapshotStore)
	if !ok {
		return nil, nil
	}
	snapshot, err := store.LatestAnyPacketSnapshotByProject(ctx, projectID)
	if err != nil {
		if isNotFound(err, "PACKET_SNAPSHOT_NOT_FOUND") {
			return nil, nil
		}
		return nil, err
	}
	return &snapshot, nil
}

func appendString(values []string, value string) []string {
	if value == "" {
		return values
	}
	return append(values, value)
}

func graphTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func projectGraphDerivedEdges(fromID string, noteIDs []string, artifactIDs []string, knownNodeIDs map[string]struct{}) []ProjectGraphEdge {
	edges := make([]ProjectGraphEdge, 0, len(noteIDs)+len(artifactIDs))

	appendEdge := func(targetID string) {
		if _, ok := knownNodeIDs[targetID]; !ok {
			return
		}
		edges = append(edges, ProjectGraphEdge{
			Type: projectGraphEdgeDerivedFrom,
			From: fromID,
			To:   targetID,
		})
	}

	for _, noteID := range noteIDs {
		appendEdge(noteID)
	}
	for _, artifactID := range artifactIDs {
		appendEdge(artifactID)
	}

	return edges
}
