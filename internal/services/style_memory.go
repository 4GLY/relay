package services

import (
	"context"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

func (s Service) WriteJudgmentTrace(ctx context.Context, input JudgmentTraceWriteInput) (JudgmentTraceWriteResult, error) {
	if err := validateJudgmentTraceWriteInput(input); err != nil {
		return JudgmentTraceWriteResult{}, err
	}
	if input.TaskID == "" || input.AgentID == "" || input.Decision == "" || input.Rationale == "" {
		return JudgmentTraceWriteResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "task_id", "agent_id", "decision", "rationale")
	}
	if input.Workflow == "" {
		input.Workflow = string(contracts.WorkflowDesignHandoff)
	}
	if input.ArtifactType == "" {
		input.ArtifactType = string(contracts.ArtifactKindDesignDoc)
	}
	if input.Language == "" {
		input.Language = "unknown"
	}
	if s.deps.JudgmentTraces == nil {
		return JudgmentTraceWriteResult{}, lib.Misconfigured("judgment trace store is required")
	}

	project, err := s.resolveProject(ctx, input.Project, input.ProjectID)
	if err != nil {
		return JudgmentTraceWriteResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return JudgmentTraceWriteResult{}, err
	}

	idempotencyPayload := input
	idempotencyPayload.IdempotencyKey = ""
	requestHash := normalizedRequestHash(idempotencyPayload)
	if lookup, err := s.lookupIdempotency(ctx, "judgment_trace", project.ID, input.IdempotencyKey, requestHash); err != nil {
		return JudgmentTraceWriteResult{}, err
	} else if lookup.found {
		trace, err := s.deps.JudgmentTraces.GetJudgmentTrace(ctx, lookup.responseID)
		if err != nil {
			return JudgmentTraceWriteResult{}, err
		}
		result := JudgmentTraceWriteResult{TraceID: trace.ID, ProjectID: trace.ProjectID}
		if s.deps.CuratorJobs != nil {
			result.CuratorJobID = curatorJudgmentTraceJobID(trace.ID)
		}
		return result, nil
	}

	traceID := lib.NewID("trace")
	if input.IdempotencyKey != "" {
		traceID = lib.StableID("trace", project.ID+":"+input.IdempotencyKey)
	}
	trace, err := s.deps.JudgmentTraces.CreateJudgmentTrace(ctx, domain.JudgmentTrace{
		ID:           traceID,
		ProjectID:    project.ID,
		TaskID:       input.TaskID,
		AgentID:      input.AgentID,
		Workflow:     input.Workflow,
		ArtifactType: input.ArtifactType,
		Decision:     input.Decision,
		Alternatives: input.Alternatives,
		Rationale:    input.Rationale,
		Constraints:  input.Constraints,
		SourceRefs:   input.SourceRefs,
		Language:     input.Language,
	})
	if err != nil {
		return JudgmentTraceWriteResult{}, err
	}
	if err := s.recordIdempotency(ctx, "judgment_trace", project.ID, input.IdempotencyKey, requestHash, "judgment_trace", trace.ID); err != nil {
		return JudgmentTraceWriteResult{}, err
	}
	job, _ := s.enqueueCuratorJudgmentTrace(ctx, trace)
	return JudgmentTraceWriteResult{TraceID: trace.ID, ProjectID: trace.ProjectID, CuratorJobID: job.ID}, nil
}

func (s Service) CreateHeuristicProposal(ctx context.Context, input HeuristicProposalCreateInput) (HeuristicProposalCreateResult, error) {
	if err := validateHeuristicProposalCreateInput(input); err != nil {
		return HeuristicProposalCreateResult{}, err
	}
	if input.HeuristicKey == "" || input.CanonicalText == "" {
		return HeuristicProposalCreateResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "heuristic_key", "canonical_text")
	}
	if s.deps.HeuristicProposals == nil {
		return HeuristicProposalCreateResult{}, lib.Misconfigured("heuristic proposal store is required")
	}

	project, err := s.resolveProject(ctx, input.Project, input.ProjectID)
	if err != nil {
		return HeuristicProposalCreateResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return HeuristicProposalCreateResult{}, err
	}

	idempotencyPayload := input
	idempotencyPayload.IdempotencyKey = ""
	requestHash := normalizedRequestHash(idempotencyPayload)
	if lookup, err := s.lookupIdempotency(ctx, "heuristic_proposal", project.ID, input.IdempotencyKey, requestHash); err != nil {
		return HeuristicProposalCreateResult{}, err
	} else if lookup.found {
		proposal, err := s.deps.HeuristicProposals.GetHeuristicProposal(ctx, lookup.responseID)
		if err != nil {
			return HeuristicProposalCreateResult{}, err
		}
		return HeuristicProposalCreateResult{ProposalID: proposal.ID, ProjectID: proposal.ProjectID, State: proposal.State}, nil
	}

	sourceTraceIDs := input.SourceTraceIDs
	if input.OriginTraceID != "" && !containsString(sourceTraceIDs, input.OriginTraceID) {
		sourceTraceIDs = append(sourceTraceIDs, input.OriginTraceID)
	}

	proposalID := lib.NewID("hprop")
	if input.IdempotencyKey != "" {
		proposalID = lib.StableID("hprop", project.ID+":"+input.IdempotencyKey)
	}
	proposal, err := s.deps.HeuristicProposals.CreateHeuristicProposal(ctx, domain.HeuristicProposal{
		ID:             proposalID,
		ProjectID:      project.ID,
		OriginTraceID:  input.OriginTraceID,
		Workflow:       input.Workflow,
		ArtifactType:   input.ArtifactType,
		HeuristicKey:   input.HeuristicKey,
		CanonicalText:  input.CanonicalText,
		NormalizedText: input.NormalizedText,
		State:          string(contracts.HeuristicStatePending),
		SourceTraceIDs: sourceTraceIDs,
		SourceRefs:     input.SourceRefs,
		ProposedBy:     input.ProposedBy,
	})
	if err != nil {
		return HeuristicProposalCreateResult{}, err
	}
	if err := s.recordIdempotency(ctx, "heuristic_proposal", project.ID, input.IdempotencyKey, requestHash, "heuristic_proposal", proposal.ID); err != nil {
		return HeuristicProposalCreateResult{}, err
	}
	return HeuristicProposalCreateResult{ProposalID: proposal.ID, ProjectID: proposal.ProjectID, State: proposal.State}, nil
}

func (s Service) ReviewHeuristicProposal(ctx context.Context, input HeuristicProposalReviewInput) (HeuristicProposalReviewResult, error) {
	if err := validateHeuristicProposalReviewInput(input); err != nil {
		return HeuristicProposalReviewResult{}, err
	}
	if input.ProposalID == "" || input.Action == "" {
		return HeuristicProposalReviewResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "proposal_id", "action")
	}
	if s.deps.HeuristicProposals == nil {
		return HeuristicProposalReviewResult{}, lib.Misconfigured("heuristic proposal store is required")
	}

	proposal, err := s.deps.HeuristicProposals.GetHeuristicProposal(ctx, input.ProposalID)
	if err != nil {
		return HeuristicProposalReviewResult{}, err
	}
	if err := s.requireAdminOrProjectOwner(ctx, proposal.ProjectID); err != nil {
		return HeuristicProposalReviewResult{}, err
	}
	if err := s.ensureProjectMatchesInput(ctx, proposal.ProjectID, input.Project, input.ProjectID); err != nil {
		return HeuristicProposalReviewResult{}, err
	}

	switch input.Action {
	case "approve":
		updated, err := s.deps.HeuristicProposals.UpdateHeuristicProposalState(ctx, proposal.ID, string(contracts.HeuristicStateApproved), input.ReviewNotes)
		if err != nil {
			return HeuristicProposalReviewResult{}, err
		}
		if s.deps.ApprovedHeuristics == nil {
			return HeuristicProposalReviewResult{}, lib.Misconfigured("approved heuristic store is required")
		}
		approved, err := s.deps.ApprovedHeuristics.CreateApprovedHeuristic(ctx, domain.ApprovedHeuristic{
			ID:               lib.StableID("heur", proposal.ProjectID+":"+proposal.HeuristicKey),
			ProjectID:        proposal.ProjectID,
			OriginProposalID: proposal.ID,
			Workflow:         proposal.Workflow,
			ArtifactType:     proposal.ArtifactType,
			HeuristicKey:     proposal.HeuristicKey,
			CanonicalText:    proposal.CanonicalText,
			State:            string(contracts.HeuristicStateApproved),
			SourceTraceIDs:   proposal.SourceTraceIDs,
			SourceRefs:       proposal.SourceRefs,
		})
		if err != nil {
			return HeuristicProposalReviewResult{}, err
		}
		return HeuristicProposalReviewResult{ProposalID: updated.ID, ApprovedHeuristicID: approved.ID, ProjectID: updated.ProjectID, State: updated.State}, nil
	case "reject":
		updated, err := s.deps.HeuristicProposals.UpdateHeuristicProposalState(ctx, proposal.ID, string(contracts.HeuristicStateRejected), input.ReviewNotes)
		if err != nil {
			return HeuristicProposalReviewResult{}, err
		}
		return HeuristicProposalReviewResult{ProposalID: updated.ID, ProjectID: updated.ProjectID, State: updated.State}, nil
	case "archive":
		updated, err := s.deps.HeuristicProposals.UpdateHeuristicProposalState(ctx, proposal.ID, string(contracts.HeuristicStateArchived), input.ReviewNotes)
		if err != nil {
			return HeuristicProposalReviewResult{}, err
		}
		return HeuristicProposalReviewResult{ProposalID: updated.ID, ProjectID: updated.ProjectID, State: updated.State}, nil
	default:
		return HeuristicProposalReviewResult{}, lib.AppError{Code: "INVALID_HEURISTIC_REVIEW_ACTION", Message: "action must be approve, reject, or archive", Retryable: false}
	}
}

func (s Service) UpdateApprovedHeuristic(ctx context.Context, input ApprovedHeuristicUpdateInput) (ApprovedHeuristicUpdateResult, error) {
	if err := validateApprovedHeuristicUpdateInput(input); err != nil {
		return ApprovedHeuristicUpdateResult{}, err
	}
	if input.HeuristicID == "" || input.Action == "" {
		return ApprovedHeuristicUpdateResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "heuristic_id", "action")
	}
	if s.deps.ApprovedHeuristics == nil {
		return ApprovedHeuristicUpdateResult{}, lib.Misconfigured("approved heuristic store is required")
	}

	heuristic, err := s.deps.ApprovedHeuristics.GetApprovedHeuristic(ctx, input.HeuristicID)
	if err != nil {
		return ApprovedHeuristicUpdateResult{}, err
	}
	if err := s.requireAdminOrProjectOwner(ctx, heuristic.ProjectID); err != nil {
		return ApprovedHeuristicUpdateResult{}, err
	}
	if err := s.ensureProjectMatchesInput(ctx, heuristic.ProjectID, input.Project, input.ProjectID); err != nil {
		return ApprovedHeuristicUpdateResult{}, err
	}

	nextState := ""
	switch input.Action {
	case "disable":
		nextState = string(contracts.HeuristicStateDisabled)
	case "archive":
		nextState = string(contracts.HeuristicStateArchived)
	case "approve", "enable":
		nextState = string(contracts.HeuristicStateApproved)
	default:
		return ApprovedHeuristicUpdateResult{}, lib.AppError{Code: "INVALID_HEURISTIC_ACTION", Message: "action must be disable, archive, approve, or enable", Retryable: false}
	}

	updated, err := s.deps.ApprovedHeuristics.UpdateApprovedHeuristicState(ctx, heuristic.ID, nextState)
	if err != nil {
		return ApprovedHeuristicUpdateResult{}, err
	}
	return ApprovedHeuristicUpdateResult{HeuristicID: updated.ID, ProjectID: updated.ProjectID, State: updated.State}, nil
}

func (s Service) ensureProjectMatchesInput(ctx context.Context, actualProjectID string, projectName string, projectID string) error {
	project, err := s.resolveProject(ctx, projectName, projectID)
	if err != nil {
		return err
	}
	if project.ID != actualProjectID {
		return lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	return s.enforceProjectAccess(ctx, actualProjectID)
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
