package services

import (
	"context"

	"relay/internal/lib"
)

const (
	projectGraphEdgeIncludes    = "includes"
	projectGraphEdgeDerivedFrom = "derived_from"
)

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

	nodes := make([]ProjectGraphNode, 0, 1+len(notes)+len(artifacts)+len(decisions)+len(questions))
	edges := make([]ProjectGraphEdge, 0, len(notes)+len(artifacts)+len(decisions)*2+len(questions)*2)
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

	return ProjectGraphResult{
		ProjectID: project.ID,
		Nodes:     nodes,
		Edges:     edges,
	}, nil
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
