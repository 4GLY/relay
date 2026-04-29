package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

type fakeJudgmentTraceStore struct {
	items map[string]domain.JudgmentTrace
}

func (s *fakeJudgmentTraceStore) CreateJudgmentTrace(_ context.Context, trace domain.JudgmentTrace) (domain.JudgmentTrace, error) {
	if s.items == nil {
		s.items = map[string]domain.JudgmentTrace{}
	}
	if existing, ok := s.items[trace.ID]; ok {
		return existing, nil
	}
	s.items[trace.ID] = trace
	return trace, nil
}

func (s *fakeJudgmentTraceStore) GetJudgmentTrace(_ context.Context, id string) (domain.JudgmentTrace, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.JudgmentTrace{}, lib.NotFound("JUDGMENT_TRACE_NOT_FOUND", "judgment trace not found")
	}
	return item, nil
}

func (s *fakeJudgmentTraceStore) ListJudgmentTracesByProject(_ context.Context, projectID string, _ int) ([]domain.JudgmentTrace, error) {
	var items []domain.JudgmentTrace
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeHeuristicProposalStore struct {
	items map[string]domain.HeuristicProposal
}

func (s *fakeHeuristicProposalStore) CreateHeuristicProposal(_ context.Context, proposal domain.HeuristicProposal) (domain.HeuristicProposal, error) {
	if s.items == nil {
		s.items = map[string]domain.HeuristicProposal{}
	}
	if existing, ok := s.items[proposal.ID]; ok {
		return existing, nil
	}
	s.items[proposal.ID] = proposal
	return proposal, nil
}

func (s *fakeHeuristicProposalStore) GetHeuristicProposal(_ context.Context, id string) (domain.HeuristicProposal, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.HeuristicProposal{}, lib.NotFound("HEURISTIC_PROPOSAL_NOT_FOUND", "heuristic proposal not found")
	}
	return item, nil
}

func (s *fakeHeuristicProposalStore) ListHeuristicProposalsByProject(_ context.Context, projectID string, state string, _ string, _ int) ([]domain.HeuristicProposal, error) {
	var items []domain.HeuristicProposal
	for _, item := range s.items {
		if item.ProjectID == projectID && (state == "" || item.State == state) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *fakeHeuristicProposalStore) UpdateHeuristicProposalState(_ context.Context, id string, state string, reviewNotes string) (domain.HeuristicProposal, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.HeuristicProposal{}, lib.NotFound("HEURISTIC_PROPOSAL_NOT_FOUND", "heuristic proposal not found")
	}
	item.State = state
	item.ReviewNotes = reviewNotes
	s.items[id] = item
	return item, nil
}

type fakeApprovedHeuristicStore struct {
	items map[string]domain.ApprovedHeuristic
}

func (s *fakeApprovedHeuristicStore) CreateApprovedHeuristic(_ context.Context, heuristic domain.ApprovedHeuristic) (domain.ApprovedHeuristic, error) {
	if s.items == nil {
		s.items = map[string]domain.ApprovedHeuristic{}
	}
	if existing, ok := s.items[heuristic.ID]; ok {
		return existing, nil
	}
	s.items[heuristic.ID] = heuristic
	return heuristic, nil
}

func (s *fakeApprovedHeuristicStore) GetApprovedHeuristic(_ context.Context, id string) (domain.ApprovedHeuristic, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.ApprovedHeuristic{}, lib.NotFound("APPROVED_HEURISTIC_NOT_FOUND", "approved heuristic not found")
	}
	return item, nil
}

func (s *fakeApprovedHeuristicStore) ListApprovedHeuristicsByProject(_ context.Context, projectID string, workflow string, artifactType string, _ string, limit int) ([]domain.ApprovedHeuristic, error) {
	var items []domain.ApprovedHeuristic
	for _, item := range s.items {
		if item.ProjectID != projectID || item.State != string(contracts.HeuristicStateApproved) {
			continue
		}
		if workflow != "" && item.Workflow != workflow {
			continue
		}
		if artifactType != "" && item.ArtifactType != artifactType {
			continue
		}
		items = append(items, item)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (s *fakeApprovedHeuristicStore) UpdateApprovedHeuristicState(_ context.Context, id string, state string) (domain.ApprovedHeuristic, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.ApprovedHeuristic{}, lib.NotFound("APPROVED_HEURISTIC_NOT_FOUND", "approved heuristic not found")
	}
	item.State = state
	s.items[id] = item
	return item, nil
}

type fakePacketSnapshotStore struct {
	items map[string]domain.PacketSnapshot
}

func (s *fakePacketSnapshotStore) CreatePacketSnapshot(_ context.Context, snapshot domain.PacketSnapshot) (domain.PacketSnapshot, error) {
	if s.items == nil {
		s.items = map[string]domain.PacketSnapshot{}
	}
	if existing, ok := s.items[snapshot.ID]; ok {
		return existing, nil
	}
	s.items[snapshot.ID] = snapshot
	return snapshot, nil
}

func (s *fakePacketSnapshotStore) GetPacketSnapshot(_ context.Context, id string) (domain.PacketSnapshot, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	return item, nil
}

func (s *fakePacketSnapshotStore) LatestPacketSnapshotByProject(_ context.Context, projectID string, packetKind string, target string) (domain.PacketSnapshot, error) {
	var latest domain.PacketSnapshot
	found := false
	for _, item := range s.items {
		if item.ProjectID != projectID || item.PacketKind != packetKind || item.Target != target {
			continue
		}
		if !found || item.CreatedAt.After(latest.CreatedAt) || (item.CreatedAt.Equal(latest.CreatedAt) && item.ID > latest.ID) {
			latest = item
			found = true
		}
	}
	if !found {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	return latest, nil
}

func (s *fakePacketSnapshotStore) LatestAnyPacketSnapshotByProject(_ context.Context, projectID string) (domain.PacketSnapshot, error) {
	var latest domain.PacketSnapshot
	found := false
	for _, item := range s.items {
		if item.ProjectID != projectID {
			continue
		}
		if !found || item.CreatedAt.After(latest.CreatedAt) || (item.CreatedAt.Equal(latest.CreatedAt) && item.ID > latest.ID) {
			latest = item
			found = true
		}
	}
	if !found {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	return latest, nil
}

func (s *fakePacketSnapshotStore) CountPacketSnapshotsByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (s *fakePacketSnapshotStore) MakePacketSnapshotPublic(_ context.Context, snapshotID string, publicToken string, ogImagePath string) (domain.PacketSnapshot, error) {
	if s.items == nil {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	item, ok := s.items[snapshotID]
	if !ok {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	item.PublicReadable = true
	item.PublicToken = publicToken
	item.OGImagePath = ogImagePath
	s.items[snapshotID] = item
	return item, nil
}

func (s *fakePacketSnapshotStore) RevokePacketSnapshotPublic(_ context.Context, snapshotID string) (domain.PacketSnapshot, error) {
	if s.items == nil {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	item, ok := s.items[snapshotID]
	if !ok {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	item.PublicReadable = false
	s.items[snapshotID] = item
	return item, nil
}

func (s *fakePacketSnapshotStore) GetPacketSnapshotByPublicToken(_ context.Context, token string) (domain.PacketSnapshot, error) {
	for _, item := range s.items {
		if item.PublicToken == token && item.PublicReadable {
			return item, nil
		}
	}
	return domain.PacketSnapshot{}, lib.NotFound("PUBLIC_SNAPSHOT_NOT_FOUND", "public snapshot not found")
}

type fakeIdempotencyStore struct {
	items map[string]domain.IdempotencyRecord
}

func (s *fakeIdempotencyStore) CreateIdempotencyRecord(_ context.Context, record domain.IdempotencyRecord) (domain.IdempotencyRecord, error) {
	if s.items == nil {
		s.items = map[string]domain.IdempotencyRecord{}
	}
	key := record.ScopeKind + ":" + record.ScopeProjectID + ":" + record.IdempotencyKey
	if existing, ok := s.items[key]; ok {
		return existing, nil
	}
	s.items[key] = record
	return record, nil
}

func (s *fakeIdempotencyStore) GetIdempotencyRecord(_ context.Context, scopeKind string, scopeProjectID string, key string) (domain.IdempotencyRecord, error) {
	item, ok := s.items[scopeKind+":"+scopeProjectID+":"+key]
	if !ok {
		return domain.IdempotencyRecord{}, lib.NotFound("IDEMPOTENCY_RECORD_NOT_FOUND", "idempotency record not found")
	}
	return item, nil
}

type fakeCuratorJobStore struct {
	items map[string]domain.CuratorJob
	order []string
}

func (s *fakeCuratorJobStore) EnqueueCuratorJob(_ context.Context, job domain.CuratorJob) (domain.CuratorJob, error) {
	if s.items == nil {
		s.items = map[string]domain.CuratorJob{}
	}
	if existing, ok := s.items[job.ID]; ok {
		return existing, nil
	}
	if job.State == "" {
		job.State = string(contracts.CuratorJobStatePending)
	}
	if job.AvailableAt.IsZero() {
		job.AvailableAt = time.Now()
	}
	s.items[job.ID] = job
	s.order = append(s.order, job.ID)
	return job, nil
}

func (s *fakeCuratorJobStore) ClaimCuratorJobs(_ context.Context, owner string, limit int, leaseUntil time.Time) ([]domain.CuratorJob, error) {
	if limit <= 0 {
		limit = 1
	}
	now := time.Now()
	var jobs []domain.CuratorJob
	for _, id := range s.order {
		job := s.items[id]
		expiredLease := job.State == string(contracts.CuratorJobStateLeased) && job.LeaseExpiresAt != nil && !job.LeaseExpiresAt.After(now)
		if !(job.State == string(contracts.CuratorJobStatePending) && !job.AvailableAt.After(now)) && !expiredLease {
			continue
		}
		job.State = string(contracts.CuratorJobStateLeased)
		job.LeaseOwner = owner
		job.LeaseExpiresAt = &leaseUntil
		job.AttemptCount++
		s.items[id] = job
		jobs = append(jobs, job)
		if len(jobs) >= limit {
			break
		}
	}
	return jobs, nil
}

func (s *fakeCuratorJobStore) CompleteCuratorJob(_ context.Context, id string) (domain.CuratorJob, error) {
	job, ok := s.items[id]
	if !ok {
		return domain.CuratorJob{}, lib.NotFound("CURATOR_JOB_NOT_FOUND", "curator job not found")
	}
	job.State = string(contracts.CuratorJobStateDone)
	job.LeaseOwner = ""
	job.LeaseExpiresAt = nil
	s.items[id] = job
	return job, nil
}

func (s *fakeCuratorJobStore) FailCuratorJob(_ context.Context, id string, retryAt time.Time, lastError string) (domain.CuratorJob, error) {
	job, ok := s.items[id]
	if !ok {
		return domain.CuratorJob{}, lib.NotFound("CURATOR_JOB_NOT_FOUND", "curator job not found")
	}
	job.State = string(contracts.CuratorJobStatePending)
	job.LastError = lastError
	job.AvailableAt = retryAt
	job.LeaseOwner = ""
	job.LeaseExpiresAt = nil
	s.items[id] = job
	return job, nil
}

func (s *fakeCuratorJobStore) MarkCuratorJobFailed(_ context.Context, id string, lastError string) (domain.CuratorJob, error) {
	job, ok := s.items[id]
	if !ok {
		return domain.CuratorJob{}, lib.NotFound("CURATOR_JOB_NOT_FOUND", "curator job not found")
	}
	job.State = string(contracts.CuratorJobStateFailed)
	job.LastError = lastError
	job.LeaseOwner = ""
	job.LeaseExpiresAt = nil
	s.items[id] = job
	return job, nil
}

type failingCuratorProvider struct{}

func (failingCuratorProvider) ProposeHeuristic(context.Context, domain.JudgmentTrace) (HeuristicProposalCreateInput, error) {
	return HeuristicProposalCreateInput{}, lib.AppError{Code: "CURATOR_PROVIDER_FAILED", Message: "provider failed", Retryable: true}
}

func TestWriteJudgmentTraceIsIdempotentAndRejectsConflictingPayload(t *testing.T) {
	projectID := lib.ProjectID("relay")
	traces := &fakeJudgmentTraceStore{}
	idempotency := &fakeIdempotencyStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		JudgmentTraces: traces,
		Idempotency:    idempotency,
	})

	input := JudgmentTraceWriteInput{
		Project:        "relay",
		TaskID:         "task-1",
		AgentID:        "agent-a",
		Decision:       "Prefer explicit contracts",
		Rationale:      "Keeps handoff deterministic",
		Language:       "ko",
		IdempotencyKey: "trace-key-1",
	}

	first, err := service.WriteJudgmentTrace(context.Background(), input)
	if err != nil {
		t.Fatalf("WriteJudgmentTrace returned error: %v", err)
	}
	second, err := service.WriteJudgmentTrace(context.Background(), input)
	if err != nil {
		t.Fatalf("WriteJudgmentTrace replay returned error: %v", err)
	}
	if second.TraceID != first.TraceID {
		t.Fatalf("expected same trace id on replay, got %q then %q", first.TraceID, second.TraceID)
	}

	input.Decision = "Prefer clever inference"
	_, err = service.WriteJudgmentTrace(context.Background(), input)
	if err == nil {
		t.Fatal("expected idempotency conflict")
	}
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("expected IDEMPOTENCY_CONFLICT, got %#v", err)
	}
}

func TestWriteJudgmentTraceEnqueuesCuratorJob(t *testing.T) {
	projectID := lib.ProjectID("relay")
	jobs := &fakeCuratorJobStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		JudgmentTraces: &fakeJudgmentTraceStore{},
		Idempotency:    &fakeIdempotencyStore{},
		CuratorJobs:    jobs,
	})

	first, err := service.WriteJudgmentTrace(context.Background(), JudgmentTraceWriteInput{
		Project:        "relay",
		TaskID:         "task-1",
		AgentID:        "codex",
		Decision:       "Prefer explicit contracts",
		Rationale:      "Keeps handoff deterministic",
		IdempotencyKey: "trace-key-1",
	})
	if err != nil {
		t.Fatalf("WriteJudgmentTrace returned error: %v", err)
	}
	second, err := service.WriteJudgmentTrace(context.Background(), JudgmentTraceWriteInput{
		Project:        "relay",
		TaskID:         "task-1",
		AgentID:        "codex",
		Decision:       "Prefer explicit contracts",
		Rationale:      "Keeps handoff deterministic",
		IdempotencyKey: "trace-key-1",
	})
	if err != nil {
		t.Fatalf("WriteJudgmentTrace replay returned error: %v", err)
	}
	if first.CuratorJobID == "" || second.CuratorJobID != first.CuratorJobID {
		t.Fatalf("expected stable curator job id, got %q then %q", first.CuratorJobID, second.CuratorJobID)
	}
	if len(jobs.items) != 1 {
		t.Fatalf("expected exactly one curator job, got %d", len(jobs.items))
	}
	job := jobs.items[first.CuratorJobID]
	if job.JobKind != string(contracts.CuratorJobKindJudgmentTrace) || job.ProjectID != projectID {
		t.Fatalf("unexpected curator job: %#v", job)
	}
}

func TestCuratorWorkerCreatesProposalOnly(t *testing.T) {
	projectID := lib.ProjectID("relay")
	traces := &fakeJudgmentTraceStore{}
	proposals := &fakeHeuristicProposalStore{}
	approved := &fakeApprovedHeuristicStore{}
	jobs := &fakeCuratorJobStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		JudgmentTraces:     traces,
		HeuristicProposals: proposals,
		ApprovedHeuristics: approved,
		Idempotency:        &fakeIdempotencyStore{},
		CuratorJobs:        jobs,
	})

	traceResult, err := service.WriteJudgmentTrace(context.Background(), JudgmentTraceWriteInput{
		Project:        "relay",
		TaskID:         "task-1",
		AgentID:        "codex",
		Decision:       "Prefer explicit contracts over magic inference",
		Rationale:      "The next model should not infer hidden product contracts.",
		IdempotencyKey: "trace-key-1",
	})
	if err != nil {
		t.Fatalf("WriteJudgmentTrace returned error: %v", err)
	}

	result, err := service.RunCuratorOnce(context.Background(), RuleBasedCuratorProvider{}, CuratorRunOptions{Owner: "worker-1"})
	if err != nil {
		t.Fatalf("RunCuratorOnce returned error: %v", err)
	}
	if result.Claimed != 1 || result.Completed != 1 || len(result.ProposalIDs) != 1 {
		t.Fatalf("unexpected curator result: %#v", result)
	}
	if jobs.items[traceResult.CuratorJobID].State != string(contracts.CuratorJobStateDone) {
		t.Fatalf("expected job done, got %#v", jobs.items[traceResult.CuratorJobID])
	}
	if len(proposals.items) != 1 {
		t.Fatalf("expected one proposal, got %d", len(proposals.items))
	}
	for _, proposal := range proposals.items {
		if proposal.State != string(contracts.HeuristicStatePending) {
			t.Fatalf("expected pending proposal, got %#v", proposal)
		}
		if proposal.ProposedBy != "curator:rule-based" {
			t.Fatalf("expected curator proposed_by, got %q", proposal.ProposedBy)
		}
	}
	if len(approved.items) != 0 {
		t.Fatalf("curator must not auto-approve heuristics: %#v", approved.items)
	}
}

func TestCuratorWorkerRetriesThenFailsJob(t *testing.T) {
	projectID := lib.ProjectID("relay")
	jobs := &fakeCuratorJobStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		JudgmentTraces: &fakeJudgmentTraceStore{items: map[string]domain.JudgmentTrace{
			"trace_1": {
				ID:        "trace_1",
				ProjectID: projectID,
				Decision:  "Prefer explicit contracts",
				Rationale: "Keeps handoff deterministic",
			},
		}},
		HeuristicProposals: &fakeHeuristicProposalStore{},
		CuratorJobs:        jobs,
	})
	job, err := service.enqueueCuratorJudgmentTrace(context.Background(), domain.JudgmentTrace{ID: "trace_1", ProjectID: projectID})
	if err != nil {
		t.Fatalf("enqueue curator job: %v", err)
	}

	first, err := service.RunCuratorOnce(context.Background(), failingCuratorProvider{}, CuratorRunOptions{
		Owner:         "worker-1",
		MaxAttempts:   2,
		RetryBackoff:  time.Millisecond,
		LeaseDuration: time.Second,
	})
	if err != nil {
		t.Fatalf("first RunCuratorOnce returned error: %v", err)
	}
	if first.Retried != 1 || jobs.items[job.ID].State != string(contracts.CuratorJobStatePending) {
		t.Fatalf("expected retry after first failure, result=%#v job=%#v", first, jobs.items[job.ID])
	}
	jobs.items[job.ID] = domain.CuratorJob{
		ID:           jobs.items[job.ID].ID,
		ProjectID:    jobs.items[job.ID].ProjectID,
		JobKind:      jobs.items[job.ID].JobKind,
		State:        string(contracts.CuratorJobStatePending),
		DedupeKey:    jobs.items[job.ID].DedupeKey,
		Payload:      jobs.items[job.ID].Payload,
		AvailableAt:  time.Now().Add(-time.Second),
		AttemptCount: jobs.items[job.ID].AttemptCount,
	}

	second, err := service.RunCuratorOnce(context.Background(), failingCuratorProvider{}, CuratorRunOptions{
		Owner:         "worker-1",
		MaxAttempts:   2,
		RetryBackoff:  time.Second,
		LeaseDuration: time.Second,
	})
	if err != nil {
		t.Fatalf("second RunCuratorOnce returned error: %v", err)
	}
	if second.Failed != 1 || jobs.items[job.ID].State != string(contracts.CuratorJobStateFailed) {
		t.Fatalf("expected failed job after max attempts, result=%#v job=%#v", second, jobs.items[job.ID])
	}
}

func TestHeuristicProposalApprovalPreservesLineage(t *testing.T) {
	projectID := lib.ProjectID("relay")
	proposals := &fakeHeuristicProposalStore{}
	approved := &fakeApprovedHeuristicStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		HeuristicProposals: proposals,
		ApprovedHeuristics: approved,
	})

	created, err := service.CreateHeuristicProposal(context.Background(), HeuristicProposalCreateInput{
		Project:        "relay",
		OriginTraceID:  "trace-1",
		Workflow:       string(contracts.WorkflowDesignHandoff),
		ArtifactType:   string(contracts.ArtifactKindDesignDoc),
		HeuristicKey:   "explicit_contracts_over_magic",
		CanonicalText:  "Prefer explicit contracts over magic inference.",
		SourceTraceIDs: []string{"trace-2"},
		SourceRefs:     []string{"docs/design.md"},
		ProposedBy:     "manual",
	})
	if err != nil {
		t.Fatalf("CreateHeuristicProposal returned error: %v", err)
	}

	if _, err := service.ReviewHeuristicProposal(context.Background(), HeuristicProposalReviewInput{
		Project:    "relay",
		ProposalID: created.ProposalID,
		Action:     "approve",
	}); err == nil {
		t.Fatal("expected non-admin review to be rejected")
	} else if appErr, ok := err.(lib.AppError); !ok || appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN for non-admin review, got %#v", err)
	}

	adminCtx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})
	reviewed, err := service.ReviewHeuristicProposal(adminCtx, HeuristicProposalReviewInput{
		Project:    "relay",
		ProposalID: created.ProposalID,
		Action:     "approve",
	})
	if err != nil {
		t.Fatalf("ReviewHeuristicProposal returned error: %v", err)
	}
	if reviewed.ApprovedHeuristicID == "" {
		t.Fatal("expected approved heuristic id")
	}
	item := approved.items[reviewed.ApprovedHeuristicID]
	if item.OriginProposalID != created.ProposalID {
		t.Fatalf("expected origin proposal %q, got %q", created.ProposalID, item.OriginProposalID)
	}
	if len(item.SourceTraceIDs) != 2 {
		t.Fatalf("expected origin trace plus source trace, got %#v", item.SourceTraceIDs)
	}
}

func TestBuildPacketIncludesApprovedStyleCueAndPersistsSnapshot(t *testing.T) {
	projectID := lib.ProjectID("relay")
	approved := &fakeApprovedHeuristicStore{items: map[string]domain.ApprovedHeuristic{
		"heur_1": {
			ID:             "heur_1",
			ProjectID:      projectID,
			Workflow:       string(contracts.WorkflowDesignHandoff),
			ArtifactType:   string(contracts.ArtifactKindDesignDoc),
			HeuristicKey:   "explicit_contracts_over_magic",
			CanonicalText:  "Prefer explicit contracts over magic inference.",
			State:          string(contracts.HeuristicStateApproved),
			SourceTraceIDs: []string{"trace-1"},
		},
	}}
	snapshots := &fakePacketSnapshotStore{}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		Notes:              &fakeNoteStore{},
		Artifacts:          &fakeArtifactStore{},
		Decisions:          &fakeDecisionStore{},
		OpenQuestions:      &fakeOpenQuestionStore{},
		Packets:            &fakePacketStore{},
		ApprovedHeuristics: approved,
		PacketSnapshots:    snapshots,
		Idempotency:        &fakeIdempotencyStore{},
	})

	result, err := service.BuildPacket(context.Background(), PacketBuildInput{
		Project:         "relay",
		Type:            string(contracts.PacketKindStyleHandoff),
		Workflow:        string(contracts.WorkflowDesignHandoff),
		ArtifactType:    string(contracts.ArtifactKindDesignDoc),
		PersistSnapshot: true,
		IdempotencyKey:  "packet-key-1",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	if result.SchemaVersion == "" {
		t.Fatal("expected schema version")
	}
	if len(result.StyleCues) != 1 {
		t.Fatalf("expected 1 style cue, got %#v", result.StyleCues)
	}
	if result.StyleCues[0].CanonicalText != "Prefer explicit contracts over magic inference." {
		t.Fatalf("expected canonical text in style cue, got %#v", result.StyleCues[0])
	}
	if len(result.WhyIncluded) == 0 {
		t.Fatalf("expected why_included reasons, got %#v", result.WhyIncluded)
	}
	if !strings.Contains(result.RenderedBody, "Style rules to preserve:") {
		t.Fatalf("expected rendered body to include style rules, got %q", result.RenderedBody)
	}
	if result.SnapshotID == "" {
		t.Fatal("expected snapshot id")
	}
	snapshot := snapshots.items[result.SnapshotID]
	if snapshot.CreatedAt.After(time.Now()) {
		t.Fatalf("unexpected future created_at: %v", snapshot.CreatedAt)
	}
	if len(snapshot.StyleCues) == 0 {
		t.Fatalf("expected snapshot to preserve style cues, got %#v", snapshot)
	}
	if len(snapshot.ApprovedHeuristicIDs) != 1 || snapshot.ApprovedHeuristicIDs[0] != "heur_1" {
		t.Fatalf("expected snapshot to preserve heuristic id, got %#v", snapshot.ApprovedHeuristicIDs)
	}
}

func TestLatestPacketSnapshotReturnsImmutablePacket(t *testing.T) {
	projectID := lib.ProjectID("relay")
	snapshots := &fakePacketSnapshotStore{items: map[string]domain.PacketSnapshot{
		"psnap_old": {
			ID:            "psnap_old",
			ProjectID:     projectID,
			PacketKind:    string(contracts.PacketKindResume),
			Target:        string(contracts.PacketTargetCodex),
			SchemaVersion: packetSchemaVersionV1,
			RenderedBody:  "old body",
			StyleCues:     []byte(`[]`),
			CreatedAt:     time.Now().Add(-time.Hour),
		},
		"psnap_new": {
			ID:                   "psnap_new",
			ProjectID:            projectID,
			PacketKind:           string(contracts.PacketKindResume),
			Target:               string(contracts.PacketTargetCodex),
			SchemaVersion:        packetSchemaVersionV1,
			TaskSummary:          "continue Relay",
			RenderedBody:         "latest body",
			StyleCues:            []byte(`[{"heuristic_id":"heur_1","why_selected":"specific","source_summary":"approved"}]`),
			SupportingNotes:      []byte(`[{"note_id":"note_1","excerpt":"handoff context"}]`),
			SupportingDecisions:  []byte(`[]`),
			SupportingQuestions:  []byte(`[]`),
			SupportingArtifacts:  []byte(`[]`),
			WhyIncluded:          []string{"latest approved packet"},
			ApprovedHeuristicIDs: []string{"heur_1"},
			DecisionIDs:          []string{"dec_1"},
			MissingContext:       []string{"none"},
			CreatedAt:            time.Now(),
		},
	}}
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"relay": {ID: projectID, Name: "relay"},
		}},
		PacketSnapshots: snapshots,
	})

	result, err := service.LatestPacketSnapshot(context.Background(), PacketSnapshotReadInput{
		Project: "relay",
		Target:  string(contracts.PacketTargetCodex),
	})
	if err != nil {
		t.Fatalf("LatestPacketSnapshot returned error: %v", err)
	}
	if result.SnapshotID != "psnap_new" || result.RenderedBody != "latest body" {
		t.Fatalf("unexpected latest snapshot: %#v", result)
	}
	if len(result.StyleCues) != 1 || result.StyleCues[0].HeuristicID != "heur_1" {
		t.Fatalf("expected decoded style cues, got %#v", result.StyleCues)
	}
	if len(result.SupportingNotes) != 1 || result.SupportingNotes[0].NoteID != "note_1" {
		t.Fatalf("expected decoded supporting notes, got %#v", result.SupportingNotes)
	}
}
