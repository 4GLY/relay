package services

import (
	"context"
	"encoding/json"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

type curatorJudgmentTracePayload struct {
	TraceID string `json:"trace_id"`
}

func (s Service) enqueueCuratorJudgmentTrace(ctx context.Context, trace domain.JudgmentTrace) (domain.CuratorJob, error) {
	if s.deps.CuratorJobs == nil {
		return domain.CuratorJob{}, nil
	}

	payload, err := json.Marshal(curatorJudgmentTracePayload{TraceID: trace.ID})
	if err != nil {
		return domain.CuratorJob{}, err
	}
	dedupeKey := curatorJudgmentTraceDedupeKey(trace.ID)
	return s.deps.CuratorJobs.EnqueueCuratorJob(ctx, domain.CuratorJob{
		ID:        curatorJudgmentTraceJobID(trace.ID),
		ProjectID: trace.ProjectID,
		JobKind:   string(contracts.CuratorJobKindJudgmentTrace),
		State:     string(contracts.CuratorJobStatePending),
		DedupeKey: dedupeKey,
		Payload:   payload,
	})
}

func curatorJudgmentTraceDedupeKey(traceID string) string {
	return string(contracts.CuratorJobKindJudgmentTrace) + ":" + traceID
}

func curatorJudgmentTraceJobID(traceID string) string {
	return lib.StableID("cjob", curatorJudgmentTraceDedupeKey(traceID))
}
