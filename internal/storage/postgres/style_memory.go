package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"relay/internal/domain"
	"relay/internal/lib"
)

func (s Stores) CreateJudgmentTrace(ctx context.Context, trace domain.JudgmentTrace) (domain.JudgmentTrace, error) {
	if trace.Language == "" {
		trace.Language = "unknown"
	}
	alternatives, _ := json.Marshal(trace.Alternatives)
	constraints, _ := json.Marshal(trace.Constraints)
	sourceRefs, _ := json.Marshal(trace.SourceRefs)
	_, err := s.db.Exec(ctx, `
		INSERT INTO judgment_traces (
			id, project_id, task_id, agent_id, workflow, artifact_type, decision,
			alternatives, rationale, constraints, source_refs, language
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10::jsonb, $11::jsonb, $12)
		ON CONFLICT (id) DO NOTHING
	`, trace.ID, trace.ProjectID, trace.TaskID, trace.AgentID, trace.Workflow, trace.ArtifactType, trace.Decision, string(alternatives), trace.Rationale, string(constraints), string(sourceRefs), trace.Language)
	if err != nil {
		return domain.JudgmentTrace{}, err
	}
	return s.GetJudgmentTrace(ctx, trace.ID)
}

func (s Stores) GetJudgmentTrace(ctx context.Context, id string) (domain.JudgmentTrace, error) {
	var trace domain.JudgmentTrace
	var alternatives, constraints, sourceRefs []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, project_id, task_id, agent_id, workflow, artifact_type, decision,
		       alternatives, rationale, constraints, source_refs, language, created_at
		FROM judgment_traces
		WHERE id = $1
	`, id).Scan(&trace.ID, &trace.ProjectID, &trace.TaskID, &trace.AgentID, &trace.Workflow, &trace.ArtifactType, &trace.Decision, &alternatives, &trace.Rationale, &constraints, &sourceRefs, &trace.Language, &trace.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.JudgmentTrace{}, lib.NotFound("JUDGMENT_TRACE_NOT_FOUND", "judgment trace not found")
	}
	if err != nil {
		return domain.JudgmentTrace{}, err
	}
	_ = json.Unmarshal(alternatives, &trace.Alternatives)
	_ = json.Unmarshal(constraints, &trace.Constraints)
	_ = json.Unmarshal(sourceRefs, &trace.SourceRefs)
	return trace, nil
}

func (s Stores) ListJudgmentTracesByProject(ctx context.Context, projectID string, limit int) ([]domain.JudgmentTrace, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, project_id, task_id, agent_id, workflow, artifact_type, decision,
		       alternatives, rationale, constraints, source_refs, language, created_at
		FROM judgment_traces
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.JudgmentTrace
	for rows.Next() {
		var item domain.JudgmentTrace
		var alternatives, constraints, sourceRefs []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.TaskID, &item.AgentID, &item.Workflow, &item.ArtifactType, &item.Decision, &alternatives, &item.Rationale, &constraints, &sourceRefs, &item.Language, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(alternatives, &item.Alternatives)
		_ = json.Unmarshal(constraints, &item.Constraints)
		_ = json.Unmarshal(sourceRefs, &item.SourceRefs)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) CreateHeuristicProposal(ctx context.Context, proposal domain.HeuristicProposal) (domain.HeuristicProposal, error) {
	if proposal.State == "" {
		proposal.State = "pending"
	}
	sourceTraceIDs, _ := json.Marshal(proposal.SourceTraceIDs)
	sourceRefs, _ := json.Marshal(proposal.SourceRefs)
	_, err := s.db.Exec(ctx, `
		INSERT INTO heuristic_proposals (
			id, project_id, origin_trace_id, workflow, artifact_type, heuristic_key,
			canonical_text, normalized_text, state, source_trace_ids, source_refs,
			proposed_by, review_notes
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13)
		ON CONFLICT (id) DO NOTHING
	`, proposal.ID, proposal.ProjectID, proposal.OriginTraceID, proposal.Workflow, proposal.ArtifactType, proposal.HeuristicKey, proposal.CanonicalText, proposal.NormalizedText, proposal.State, string(sourceTraceIDs), string(sourceRefs), proposal.ProposedBy, proposal.ReviewNotes)
	if err != nil {
		return domain.HeuristicProposal{}, err
	}
	return s.GetHeuristicProposal(ctx, proposal.ID)
}

func (s Stores) GetHeuristicProposal(ctx context.Context, id string) (domain.HeuristicProposal, error) {
	var proposal domain.HeuristicProposal
	var sourceTraceIDs, sourceRefs []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, project_id, COALESCE(origin_trace_id, ''), workflow, artifact_type,
		       heuristic_key, canonical_text, normalized_text, state, source_trace_ids,
		       source_refs, proposed_by, review_notes, created_at, updated_at
		FROM heuristic_proposals
		WHERE id = $1
	`, id).Scan(&proposal.ID, &proposal.ProjectID, &proposal.OriginTraceID, &proposal.Workflow, &proposal.ArtifactType, &proposal.HeuristicKey, &proposal.CanonicalText, &proposal.NormalizedText, &proposal.State, &sourceTraceIDs, &sourceRefs, &proposal.ProposedBy, &proposal.ReviewNotes, &proposal.CreatedAt, &proposal.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.HeuristicProposal{}, lib.NotFound("HEURISTIC_PROPOSAL_NOT_FOUND", "heuristic proposal not found")
	}
	if err != nil {
		return domain.HeuristicProposal{}, err
	}
	_ = json.Unmarshal(sourceTraceIDs, &proposal.SourceTraceIDs)
	_ = json.Unmarshal(sourceRefs, &proposal.SourceRefs)
	return proposal, nil
}

func (s Stores) ListHeuristicProposalsByProject(ctx context.Context, projectID string, state string, limit int) ([]domain.HeuristicProposal, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id
		FROM heuristic_proposals
		WHERE project_id = $1
		  AND ($2 = '' OR state = $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, projectID, state, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]domain.HeuristicProposal, 0, len(ids))
	for _, id := range ids {
		item, err := s.GetHeuristicProposal(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s Stores) UpdateHeuristicProposalState(ctx context.Context, id string, state string, reviewNotes string) (domain.HeuristicProposal, error) {
	err := s.db.QueryRow(ctx, `
		UPDATE heuristic_proposals
		SET state = $2,
		    review_notes = $3,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id, state, reviewNotes).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.HeuristicProposal{}, lib.NotFound("HEURISTIC_PROPOSAL_NOT_FOUND", "heuristic proposal not found")
	}
	if err != nil {
		return domain.HeuristicProposal{}, err
	}
	return s.GetHeuristicProposal(ctx, id)
}

func (s Stores) CreateApprovedHeuristic(ctx context.Context, heuristic domain.ApprovedHeuristic) (domain.ApprovedHeuristic, error) {
	if heuristic.State == "" {
		heuristic.State = "approved"
	}
	sourceTraceIDs, _ := json.Marshal(heuristic.SourceTraceIDs)
	sourceRefs, _ := json.Marshal(heuristic.SourceRefs)
	_, err := s.db.Exec(ctx, `
		INSERT INTO approved_heuristics (
			id, project_id, origin_proposal_id, workflow, artifact_type, heuristic_key,
			canonical_text, state, source_trace_ids, source_refs
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, heuristic.ID, heuristic.ProjectID, heuristic.OriginProposalID, heuristic.Workflow, heuristic.ArtifactType, heuristic.HeuristicKey, heuristic.CanonicalText, heuristic.State, string(sourceTraceIDs), string(sourceRefs))
	if err != nil {
		return domain.ApprovedHeuristic{}, err
	}
	return s.GetApprovedHeuristic(ctx, heuristic.ID)
}

func (s Stores) GetApprovedHeuristic(ctx context.Context, id string) (domain.ApprovedHeuristic, error) {
	var heuristic domain.ApprovedHeuristic
	var sourceTraceIDs, sourceRefs []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, project_id, COALESCE(origin_proposal_id, ''), workflow, artifact_type,
		       heuristic_key, canonical_text, state, source_trace_ids, source_refs,
		       created_at, updated_at
		FROM approved_heuristics
		WHERE id = $1
	`, id).Scan(&heuristic.ID, &heuristic.ProjectID, &heuristic.OriginProposalID, &heuristic.Workflow, &heuristic.ArtifactType, &heuristic.HeuristicKey, &heuristic.CanonicalText, &heuristic.State, &sourceTraceIDs, &sourceRefs, &heuristic.CreatedAt, &heuristic.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ApprovedHeuristic{}, lib.NotFound("APPROVED_HEURISTIC_NOT_FOUND", "approved heuristic not found")
	}
	if err != nil {
		return domain.ApprovedHeuristic{}, err
	}
	_ = json.Unmarshal(sourceTraceIDs, &heuristic.SourceTraceIDs)
	_ = json.Unmarshal(sourceRefs, &heuristic.SourceRefs)
	return heuristic, nil
}

func (s Stores) ListApprovedHeuristicsByProject(ctx context.Context, projectID string, workflow string, artifactType string, limit int) ([]domain.ApprovedHeuristic, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id
		FROM approved_heuristics
		WHERE project_id = $1
		  AND state = 'approved'
		  AND (workflow = '' OR workflow = $2)
		  AND (artifact_type = '' OR artifact_type = $3)
		ORDER BY created_at DESC
		LIMIT $4
	`, projectID, workflow, artifactType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]domain.ApprovedHeuristic, 0, len(ids))
	for _, id := range ids {
		item, err := s.GetApprovedHeuristic(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s Stores) UpdateApprovedHeuristicState(ctx context.Context, id string, state string) (domain.ApprovedHeuristic, error) {
	err := s.db.QueryRow(ctx, `
		UPDATE approved_heuristics
		SET state = $2,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id, state).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ApprovedHeuristic{}, lib.NotFound("APPROVED_HEURISTIC_NOT_FOUND", "approved heuristic not found")
	}
	if err != nil {
		return domain.ApprovedHeuristic{}, err
	}
	return s.GetApprovedHeuristic(ctx, id)
}

func (s Stores) CreatePacketSnapshot(ctx context.Context, snapshot domain.PacketSnapshot) (domain.PacketSnapshot, error) {
	heuristicIDs, _ := json.Marshal(snapshot.ApprovedHeuristicIDs)
	decisionIDs, _ := json.Marshal(snapshot.DecisionIDs)
	openQuestionIDs, _ := json.Marshal(snapshot.OpenQuestionIDs)
	artifactIDs, _ := json.Marshal(snapshot.SourceArtifactIDs)
	missingContext, _ := json.Marshal(snapshot.MissingContext)
	styleCues := snapshot.StyleCues
	if len(styleCues) == 0 {
		styleCues = []byte("[]")
	}
	supportingNotes := snapshot.SupportingNotes
	if len(supportingNotes) == 0 {
		supportingNotes = []byte("[]")
	}
	supportingDecisions := snapshot.SupportingDecisions
	if len(supportingDecisions) == 0 {
		supportingDecisions = []byte("[]")
	}
	supportingQuestions := snapshot.SupportingQuestions
	if len(supportingQuestions) == 0 {
		supportingQuestions = []byte("[]")
	}
	supportingArtifacts := snapshot.SupportingArtifacts
	if len(supportingArtifacts) == 0 {
		supportingArtifacts = []byte("[]")
	}
	whyIncluded, _ := json.Marshal(snapshot.WhyIncluded)
	_, err := s.db.Exec(ctx, `
		INSERT INTO packet_snapshots (
			id, project_id, packet_kind, target, schema_version, task_summary,
			rendered_body, style_cues, supporting_notes, supporting_decisions,
			supporting_questions, supporting_artifacts, why_included,
			approved_heuristic_ids, decision_ids, open_question_ids,
			source_artifact_ids, missing_context, public_readable, public_token,
			og_image_path
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13::jsonb, $14::jsonb, $15::jsonb, $16::jsonb, $17::jsonb, $18::jsonb, $19, NULLIF($20, ''), $21)
		ON CONFLICT (id) DO NOTHING
	`, snapshot.ID, snapshot.ProjectID, snapshot.PacketKind, snapshot.Target, snapshot.SchemaVersion, snapshot.TaskSummary, snapshot.RenderedBody, string(styleCues), string(supportingNotes), string(supportingDecisions), string(supportingQuestions), string(supportingArtifacts), string(whyIncluded), string(heuristicIDs), string(decisionIDs), string(openQuestionIDs), string(artifactIDs), string(missingContext), snapshot.PublicReadable, snapshot.PublicToken, snapshot.OGImagePath)
	if err != nil {
		return domain.PacketSnapshot{}, err
	}
	return s.GetPacketSnapshot(ctx, snapshot.ID)
}

func (s Stores) GetPacketSnapshot(ctx context.Context, id string) (domain.PacketSnapshot, error) {
	return scanPacketSnapshot(s.db.QueryRow(ctx, `
		SELECT id, project_id, packet_kind, target, schema_version, task_summary,
		       rendered_body, style_cues, supporting_notes, supporting_decisions,
		       supporting_questions, supporting_artifacts, why_included,
		       approved_heuristic_ids, decision_ids, open_question_ids,
		       source_artifact_ids, missing_context, public_readable,
		       COALESCE(public_token, ''), og_image_path, created_at
		FROM packet_snapshots
		WHERE id = $1
	`, id))
}

func (s Stores) LatestPacketSnapshotByProject(ctx context.Context, projectID string, packetKind string, target string) (domain.PacketSnapshot, error) {
	return scanPacketSnapshot(s.db.QueryRow(ctx, `
		SELECT id, project_id, packet_kind, target, schema_version, task_summary,
		       rendered_body, style_cues, supporting_notes, supporting_decisions,
		       supporting_questions, supporting_artifacts, why_included,
		       approved_heuristic_ids, decision_ids, open_question_ids,
		       source_artifact_ids, missing_context, public_readable,
		       COALESCE(public_token, ''), og_image_path, created_at
		FROM packet_snapshots
		WHERE project_id = $1
		  AND packet_kind = $2
		  AND target = $3
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, projectID, packetKind, target))
}

func scanPacketSnapshot(row pgx.Row) (domain.PacketSnapshot, error) {
	var snapshot domain.PacketSnapshot
	var heuristicIDs, decisionIDs, openQuestionIDs, artifactIDs, missingContext, whyIncluded []byte
	err := row.Scan(&snapshot.ID, &snapshot.ProjectID, &snapshot.PacketKind, &snapshot.Target, &snapshot.SchemaVersion, &snapshot.TaskSummary, &snapshot.RenderedBody, &snapshot.StyleCues, &snapshot.SupportingNotes, &snapshot.SupportingDecisions, &snapshot.SupportingQuestions, &snapshot.SupportingArtifacts, &whyIncluded, &heuristicIDs, &decisionIDs, &openQuestionIDs, &artifactIDs, &missingContext, &snapshot.PublicReadable, &snapshot.PublicToken, &snapshot.OGImagePath, &snapshot.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
	}
	if err != nil {
		return domain.PacketSnapshot{}, err
	}
	_ = json.Unmarshal(whyIncluded, &snapshot.WhyIncluded)
	_ = json.Unmarshal(heuristicIDs, &snapshot.ApprovedHeuristicIDs)
	_ = json.Unmarshal(decisionIDs, &snapshot.DecisionIDs)
	_ = json.Unmarshal(openQuestionIDs, &snapshot.OpenQuestionIDs)
	_ = json.Unmarshal(artifactIDs, &snapshot.SourceArtifactIDs)
	_ = json.Unmarshal(missingContext, &snapshot.MissingContext)
	return snapshot, nil
}

func (s Stores) CreateIdempotencyRecord(ctx context.Context, record domain.IdempotencyRecord) (domain.IdempotencyRecord, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO idempotency_records (
			id, scope_kind, scope_project_id, idempotency_key, request_hash,
			response_kind, response_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (scope_kind, scope_project_id, idempotency_key) DO NOTHING
	`, record.ID, record.ScopeKind, record.ScopeProjectID, record.IdempotencyKey, record.RequestHash, record.ResponseKind, record.ResponseID)
	if err != nil {
		return domain.IdempotencyRecord{}, err
	}
	return s.GetIdempotencyRecord(ctx, record.ScopeKind, record.ScopeProjectID, record.IdempotencyKey)
}

func (s Stores) GetIdempotencyRecord(ctx context.Context, scopeKind string, scopeProjectID string, key string) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	err := s.db.QueryRow(ctx, `
		SELECT id, scope_kind, scope_project_id, idempotency_key, request_hash,
		       response_kind, response_id, created_at
		FROM idempotency_records
		WHERE scope_kind = $1
		  AND scope_project_id = $2
		  AND idempotency_key = $3
	`, scopeKind, scopeProjectID, key).Scan(&record.ID, &record.ScopeKind, &record.ScopeProjectID, &record.IdempotencyKey, &record.RequestHash, &record.ResponseKind, &record.ResponseID, &record.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.IdempotencyRecord{}, lib.NotFound("IDEMPOTENCY_RECORD_NOT_FOUND", "idempotency record not found")
	}
	return record, err
}

func (s Stores) EnqueueCuratorJob(ctx context.Context, job domain.CuratorJob) (domain.CuratorJob, error) {
	if job.State == "" {
		job.State = "pending"
	}
	if len(job.Payload) == 0 {
		job.Payload = []byte("{}")
	}
	if job.AvailableAt.IsZero() {
		job.AvailableAt = time.Now()
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO curator_jobs (
			id, project_id, job_kind, state, dedupe_key, payload, available_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		ON CONFLICT (id) DO NOTHING
	`, job.ID, job.ProjectID, job.JobKind, job.State, job.DedupeKey, string(job.Payload), job.AvailableAt)
	if err != nil {
		return domain.CuratorJob{}, err
	}
	return s.getCuratorJob(ctx, job.ID)
}

func (s Stores) ClaimCuratorJobs(ctx context.Context, owner string, limit int, leaseUntil time.Time) ([]domain.CuratorJob, error) {
	if limit <= 0 {
		limit = 1
	}
	rows, err := s.db.Query(ctx, `
		WITH claimable AS (
		  SELECT id
		  FROM curator_jobs
		  WHERE (state = 'pending' AND available_at <= NOW())
		     OR (state = 'leased' AND lease_expires_at IS NOT NULL AND lease_expires_at <= NOW())
		  ORDER BY created_at ASC
		  LIMIT $1
		  FOR UPDATE SKIP LOCKED
		)
		UPDATE curator_jobs
		SET state = 'leased',
		    lease_owner = $2,
		    lease_expires_at = $3,
		    attempt_count = attempt_count + 1,
		    updated_at = NOW()
		WHERE id IN (SELECT id FROM claimable)
		RETURNING id
	`, limit, owner, leaseUntil)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]domain.CuratorJob, 0, len(ids))
	for _, id := range ids {
		item, err := s.getCuratorJob(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s Stores) CompleteCuratorJob(ctx context.Context, id string) (domain.CuratorJob, error) {
	err := s.db.QueryRow(ctx, `
		UPDATE curator_jobs
		SET state = 'done',
		    lease_owner = '',
		    lease_expires_at = NULL,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CuratorJob{}, lib.NotFound("CURATOR_JOB_NOT_FOUND", "curator job not found")
	}
	if err != nil {
		return domain.CuratorJob{}, err
	}
	return s.getCuratorJob(ctx, id)
}

func (s Stores) FailCuratorJob(ctx context.Context, id string, retryAt time.Time, lastError string) (domain.CuratorJob, error) {
	err := s.db.QueryRow(ctx, `
		UPDATE curator_jobs
		SET state = 'pending',
		    last_error = $2,
		    lease_owner = '',
		    lease_expires_at = NULL,
		    available_at = $3,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id, lastError, retryAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CuratorJob{}, lib.NotFound("CURATOR_JOB_NOT_FOUND", "curator job not found")
	}
	if err != nil {
		return domain.CuratorJob{}, err
	}
	return s.getCuratorJob(ctx, id)
}

func (s Stores) MarkCuratorJobFailed(ctx context.Context, id string, lastError string) (domain.CuratorJob, error) {
	err := s.db.QueryRow(ctx, `
		UPDATE curator_jobs
		SET state = 'failed',
		    last_error = $2,
		    lease_owner = '',
		    lease_expires_at = NULL,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id, lastError).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CuratorJob{}, lib.NotFound("CURATOR_JOB_NOT_FOUND", "curator job not found")
	}
	if err != nil {
		return domain.CuratorJob{}, err
	}
	return s.getCuratorJob(ctx, id)
}

func (s Stores) getCuratorJob(ctx context.Context, id string) (domain.CuratorJob, error) {
	var job domain.CuratorJob
	var leaseExpiresAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT id, project_id, job_kind, state, dedupe_key, payload,
		       attempt_count, last_error, lease_owner, lease_expires_at,
		       available_at, created_at, updated_at
		FROM curator_jobs
		WHERE id = $1
	`, id).Scan(&job.ID, &job.ProjectID, &job.JobKind, &job.State, &job.DedupeKey, &job.Payload, &job.AttemptCount, &job.LastError, &job.LeaseOwner, &leaseExpiresAt, &job.AvailableAt, &job.CreatedAt, &job.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CuratorJob{}, lib.NotFound("CURATOR_JOB_NOT_FOUND", "curator job not found")
	}
	if err != nil {
		return domain.CuratorJob{}, err
	}
	job.LeaseExpiresAt = leaseExpiresAt
	return job, nil
}
