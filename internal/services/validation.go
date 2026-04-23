package services

import (
	"fmt"
	"unicode/utf8"

	"relay/internal/lib"
)

const (
	maxCaptureProjectLength     = 128
	maxCapturePathLength        = 2048
	maxCaptureTextLength        = 8192
	maxCaptureSourceLength      = 64
	maxCaptureIdempotencyLength = 128

	maxPromoteProjectLength     = 128
	maxPromoteKindLength        = 32
	maxPromoteSummaryLength     = 8192
	maxPromoteReasonLength      = 8192
	maxPromoteIdempotencyLength = 128
	maxPromoteSourceIDs         = 100
	maxPromoteSourceIDLength    = 128

	maxPacketProjectLength = 128
	maxPacketTypeLength    = 32
	maxPacketTargetLength  = 128
	maxPacketTaskLength    = 8192

	maxStyleProjectLength      = 128
	maxStyleIDLength           = 128
	maxStyleWorkflowLength     = 64
	maxStyleArtifactTypeLength = 64
	maxStyleTextLength         = 8192
	maxStyleLanguageLength     = 32
	maxStyleIdempotencyLength  = 128
	maxStyleSourceRefs         = 100
	maxStyleSourceRefLength    = 512
	maxStyleHeuristicKeyLength = 128
	maxStyleReviewNotesLength  = 8192
	maxStyleReviewActionLength = 32

	maxAPIKeyNameLength      = 128
	maxAPIKeyScopeLength     = 32
	maxAPIKeyProjectLength   = 128
	maxAPIKeyProjectIDLength = 128
	maxAPIKeyIDLength        = 128
)

func validateStringFieldLength(field string, value string, max int) error {
	if value == "" || utf8.RuneCountInString(value) <= max {
		return nil
	}
	return lib.AppError{
		Code:      "FIELD_TOO_LONG",
		Message:   fmt.Sprintf("%s exceeds maximum length of %d characters", field, max),
		Retryable: false,
	}
}

func validateStringSliceField(field string, values []string, maxItems int, maxItemLength int) error {
	if len(values) > maxItems {
		return lib.AppError{
			Code:      "FIELD_TOO_MANY_ITEMS",
			Message:   fmt.Sprintf("%s exceeds maximum item count of %d", field, maxItems),
			Retryable: false,
		}
	}

	for idx, value := range values {
		if err := validateStringFieldLength(fmt.Sprintf("%s[%d]", field, idx), value, maxItemLength); err != nil {
			return err
		}
	}

	return nil
}

func validateCaptureInput(input CaptureInput) error {
	if err := validateStringFieldLength("project", input.Project, maxCaptureProjectLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("repo_path", input.RepoPath, maxCapturePathLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("handoff_path", input.HandoffPath, maxCapturePathLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("design_path", input.DesignPath, maxCapturePathLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("note", input.Note, maxCaptureTextLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("source", input.Source, maxCaptureSourceLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("body", input.Body, maxCaptureTextLength); err != nil {
		return err
	}
	return validateStringFieldLength("idempotency_key", input.IdempotencyKey, maxCaptureIdempotencyLength)
}

func validatePromoteInput(input PromoteInput) error {
	if err := validateStringFieldLength("project", input.Project, maxPromoteProjectLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("kind", input.Kind, maxPromoteKindLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("summary", input.Summary, maxPromoteSummaryLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("reason", input.Reason, maxPromoteReasonLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("idempotency_key", input.IdempotencyKey, maxPromoteIdempotencyLength); err != nil {
		return err
	}
	if err := validateStringSliceField("source_note_ids", input.SourceNoteIDs, maxPromoteSourceIDs, maxPromoteSourceIDLength); err != nil {
		return err
	}
	return validateStringSliceField("source_artifact_ids", input.SourceArtifactIDs, maxPromoteSourceIDs, maxPromoteSourceIDLength)
}

func validatePacketBuildInput(input PacketBuildInput) error {
	if err := validateStringFieldLength("project", input.Project, maxPacketProjectLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("type", input.Type, maxPacketTypeLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("target", input.Target, maxPacketTargetLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("workflow", input.Workflow, maxStyleWorkflowLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("artifact_type", input.ArtifactType, maxStyleArtifactTypeLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("task_summary", input.TaskSummary, maxPacketTaskLength); err != nil {
		return err
	}
	return validateStringFieldLength("idempotency_key", input.IdempotencyKey, maxStyleIdempotencyLength)
}

func validateJudgmentTraceWriteInput(input JudgmentTraceWriteInput) error {
	if err := validateStringFieldLength("project", input.Project, maxStyleProjectLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("project_id", input.ProjectID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("task_id", input.TaskID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("agent_id", input.AgentID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("workflow", input.Workflow, maxStyleWorkflowLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("artifact_type", input.ArtifactType, maxStyleArtifactTypeLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("decision", input.Decision, maxStyleTextLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("rationale", input.Rationale, maxStyleTextLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("language", input.Language, maxStyleLanguageLength); err != nil {
		return err
	}
	if err := validateStringSliceField("alternatives", input.Alternatives, maxStyleSourceRefs, maxStyleTextLength); err != nil {
		return err
	}
	if err := validateStringSliceField("constraints", input.Constraints, maxStyleSourceRefs, maxStyleTextLength); err != nil {
		return err
	}
	if err := validateStringSliceField("source_refs", input.SourceRefs, maxStyleSourceRefs, maxStyleSourceRefLength); err != nil {
		return err
	}
	return validateStringFieldLength("idempotency_key", input.IdempotencyKey, maxStyleIdempotencyLength)
}

func validateHeuristicProposalCreateInput(input HeuristicProposalCreateInput) error {
	if err := validateStringFieldLength("project", input.Project, maxStyleProjectLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("project_id", input.ProjectID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("origin_trace_id", input.OriginTraceID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("workflow", input.Workflow, maxStyleWorkflowLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("artifact_type", input.ArtifactType, maxStyleArtifactTypeLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("heuristic_key", input.HeuristicKey, maxStyleHeuristicKeyLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("canonical_text", input.CanonicalText, maxStyleTextLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("normalized_text", input.NormalizedText, maxStyleTextLength); err != nil {
		return err
	}
	if err := validateStringSliceField("source_trace_ids", input.SourceTraceIDs, maxStyleSourceRefs, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringSliceField("source_refs", input.SourceRefs, maxStyleSourceRefs, maxStyleSourceRefLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("proposed_by", input.ProposedBy, maxStyleIDLength); err != nil {
		return err
	}
	return validateStringFieldLength("idempotency_key", input.IdempotencyKey, maxStyleIdempotencyLength)
}

func validateHeuristicProposalReviewInput(input HeuristicProposalReviewInput) error {
	if err := validateStringFieldLength("project", input.Project, maxStyleProjectLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("project_id", input.ProjectID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("proposal_id", input.ProposalID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("action", input.Action, maxStyleReviewActionLength); err != nil {
		return err
	}
	return validateStringFieldLength("review_notes", input.ReviewNotes, maxStyleReviewNotesLength)
}

func validateApprovedHeuristicUpdateInput(input ApprovedHeuristicUpdateInput) error {
	if err := validateStringFieldLength("project", input.Project, maxStyleProjectLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("project_id", input.ProjectID, maxStyleIDLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("heuristic_id", input.HeuristicID, maxStyleIDLength); err != nil {
		return err
	}
	return validateStringFieldLength("action", input.Action, maxStyleReviewActionLength)
}

func validateIssueAPIKeyInput(input IssueAPIKeyInput) error {
	if err := validateStringFieldLength("name", input.Name, maxAPIKeyNameLength); err != nil {
		return err
	}
	if err := validateStringFieldLength("scope", input.Scope, maxAPIKeyScopeLength); err != nil {
		return err
	}
	if (input.Project != "" || input.ProjectID != "") && NormalizeAPIKeyScope(input.Scope) != APIKeyScopeProject {
		return lib.AppError{
			Code:      "INVALID_API_KEY_SCOPE",
			Message:   "project and project_id require scope project",
			Retryable: false,
		}
	}
	if err := validateStringFieldLength("project", input.Project, maxAPIKeyProjectLength); err != nil {
		return err
	}
	return validateStringFieldLength("project_id", input.ProjectID, maxAPIKeyProjectIDLength)
}

func validateRevokeAPIKeyInput(input RevokeAPIKeyInput) error {
	return validateStringFieldLength("key_id", input.KeyID, maxAPIKeyIDLength)
}
