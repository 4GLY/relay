package services

import (
	"context"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

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
	case string(contracts.PromotionKindDecision):
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
	case string(contracts.PromotionKindQuestion):
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

func promotedID(prefix string, idempotencyKey string, summary string) string {
	if idempotencyKey != "" {
		return lib.StableID(prefix, "promote:"+idempotencyKey)
	}
	return lib.StableID(prefix, summary)
}
