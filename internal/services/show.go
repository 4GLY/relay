package services

import (
	"context"

	"relay/internal/lib"
)

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
