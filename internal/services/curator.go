package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

const (
	defaultCuratorOwner         = "relay-curator"
	defaultCuratorBatchSize     = 5
	defaultCuratorLeaseDuration = 30 * time.Second
	defaultCuratorRetryBackoff  = 30 * time.Second
	defaultCuratorMaxAttempts   = 5
)

type CuratorProvider interface {
	ProposeHeuristic(ctx context.Context, trace domain.JudgmentTrace) (HeuristicProposalCreateInput, error)
}

type RuleBasedCuratorProvider struct{}

func (RuleBasedCuratorProvider) ProposeHeuristic(_ context.Context, trace domain.JudgmentTrace) (HeuristicProposalCreateInput, error) {
	decision := strings.TrimSpace(trace.Decision)
	if decision == "" {
		return HeuristicProposalCreateInput{}, lib.AppError{Code: "CURATOR_EMPTY_DECISION", Message: "judgment trace decision is empty", Retryable: false}
	}

	canonicalText := decision
	if rationale := strings.TrimSpace(trace.Rationale); rationale != "" {
		canonicalText = fmt.Sprintf("%s Rationale: %s", strings.TrimRight(decision, ".!?"), rationale)
	}

	return HeuristicProposalCreateInput{
		ProjectID:      trace.ProjectID,
		OriginTraceID:  trace.ID,
		Workflow:       trace.Workflow,
		ArtifactType:   trace.ArtifactType,
		HeuristicKey:   heuristicKeyFromTrace(trace),
		CanonicalText:  canonicalText,
		NormalizedText: strings.ToLower(canonicalText),
		SourceTraceIDs: []string{trace.ID},
		SourceRefs:     trace.SourceRefs,
		ProposedBy:     "curator:rule-based",
		IdempotencyKey: "curator:" + string(contracts.CuratorJobKindJudgmentTrace) + ":" + trace.ID,
	}, nil
}

func (s Service) RunCuratorOnce(ctx context.Context, provider CuratorProvider, options CuratorRunOptions) (CuratorRunResult, error) {
	if s.deps.CuratorJobs == nil {
		return CuratorRunResult{}, lib.Misconfigured("curator job store is required")
	}
	if provider == nil {
		provider = RuleBasedCuratorProvider{}
	}

	options = normalizeCuratorOptions(options)
	jobs, err := s.deps.CuratorJobs.ClaimCuratorJobs(ctx, options.Owner, options.BatchSize, time.Now().Add(options.LeaseDuration))
	if err != nil {
		return CuratorRunResult{}, err
	}

	result := CuratorRunResult{Claimed: len(jobs)}
	for _, job := range jobs {
		result.ProcessedIDs = append(result.ProcessedIDs, job.ID)
		proposalID, err := s.processCuratorJob(ctx, provider, job)
		if err != nil {
			if failErr := s.failCuratorJob(ctx, job, options, err); failErr != nil {
				return result, failErr
			}
			if job.AttemptCount >= options.MaxAttempts {
				result.Failed++
			} else {
				result.Retried++
			}
			continue
		}
		if _, err := s.deps.CuratorJobs.CompleteCuratorJob(ctx, job.ID); err != nil {
			return result, err
		}
		result.Completed++
		result.ProposalIDs = append(result.ProposalIDs, proposalID)
	}

	return result, nil
}

func (s Service) processCuratorJob(ctx context.Context, provider CuratorProvider, job domain.CuratorJob) (string, error) {
	switch job.JobKind {
	case string(contracts.CuratorJobKindJudgmentTrace):
		return s.processCuratorJudgmentTrace(ctx, provider, job)
	default:
		return "", lib.AppError{Code: "UNKNOWN_CURATOR_JOB_KIND", Message: "unknown curator job kind: " + job.JobKind, Retryable: false}
	}
}

func (s Service) processCuratorJudgmentTrace(ctx context.Context, provider CuratorProvider, job domain.CuratorJob) (string, error) {
	if s.deps.JudgmentTraces == nil {
		return "", lib.Misconfigured("judgment trace store is required")
	}

	var payload curatorJudgmentTracePayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return "", err
	}
	if payload.TraceID == "" {
		return "", lib.MissingFields("MISSING_REQUIRED_FIELDS", "trace_id")
	}

	trace, err := s.deps.JudgmentTraces.GetJudgmentTrace(ctx, payload.TraceID)
	if err != nil {
		return "", err
	}
	if trace.ProjectID != job.ProjectID {
		return "", lib.Forbidden("FORBIDDEN", "curator job project does not match judgment trace project")
	}

	proposal, err := provider.ProposeHeuristic(ctx, trace)
	if err != nil {
		return "", err
	}
	proposal.ProjectID = trace.ProjectID
	if proposal.OriginTraceID == "" {
		proposal.OriginTraceID = trace.ID
	}
	if proposal.Workflow == "" {
		proposal.Workflow = trace.Workflow
	}
	if proposal.ArtifactType == "" {
		proposal.ArtifactType = trace.ArtifactType
	}
	if proposal.ProposedBy == "" {
		proposal.ProposedBy = "curator"
	}
	if proposal.IdempotencyKey == "" {
		proposal.IdempotencyKey = "curator:" + job.DedupeKey
	}
	if !containsString(proposal.SourceTraceIDs, trace.ID) {
		proposal.SourceTraceIDs = append(proposal.SourceTraceIDs, trace.ID)
	}

	created, err := s.CreateHeuristicProposal(ctx, proposal)
	if err != nil {
		return "", err
	}
	return created.ProposalID, nil
}

func (s Service) failCuratorJob(ctx context.Context, job domain.CuratorJob, options CuratorRunOptions, cause error) error {
	message := cause.Error()
	if job.AttemptCount >= options.MaxAttempts {
		_, err := s.deps.CuratorJobs.MarkCuratorJobFailed(ctx, job.ID, message)
		return err
	}
	retryAt := time.Now().Add(curatorRetryBackoff(options.RetryBackoff, job.AttemptCount))
	_, err := s.deps.CuratorJobs.FailCuratorJob(ctx, job.ID, retryAt, message)
	return err
}

func normalizeCuratorOptions(options CuratorRunOptions) CuratorRunOptions {
	if options.Owner == "" {
		options.Owner = defaultCuratorOwner
	}
	if options.BatchSize <= 0 {
		options.BatchSize = defaultCuratorBatchSize
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = defaultCuratorLeaseDuration
	}
	if options.RetryBackoff <= 0 {
		options.RetryBackoff = defaultCuratorRetryBackoff
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = defaultCuratorMaxAttempts
	}
	return options
}

func curatorRetryBackoff(base time.Duration, attempt int) time.Duration {
	if attempt <= 1 {
		return base
	}
	multiplier := 1 << minInt(attempt-1, 4)
	return time.Duration(multiplier) * base
}

func heuristicKeyFromTrace(trace domain.JudgmentTrace) string {
	slug := slugifyHeuristicKey(trace.Decision)
	if slug == "" {
		slug = "judgment"
	}
	if len(slug) > 56 {
		slug = strings.Trim(slug[:56], "_")
	}
	return slug + "_" + lib.StableID("h", trace.ID)[2:]
}

func slugifyHeuristicKey(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if r <= unicode.MaxASCII {
				builder.WriteRune(r)
				lastUnderscore = false
			}
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}
