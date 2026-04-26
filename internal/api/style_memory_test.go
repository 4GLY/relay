package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/services"
)

type apiFakeJudgmentTraceStore struct {
	items map[string]domain.JudgmentTrace
}

func (s *apiFakeJudgmentTraceStore) CreateJudgmentTrace(_ context.Context, trace domain.JudgmentTrace) (domain.JudgmentTrace, error) {
	if s.items == nil {
		s.items = map[string]domain.JudgmentTrace{}
	}
	if existing, ok := s.items[trace.ID]; ok {
		return existing, nil
	}
	s.items[trace.ID] = trace
	return trace, nil
}

func (s *apiFakeJudgmentTraceStore) GetJudgmentTrace(_ context.Context, id string) (domain.JudgmentTrace, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.JudgmentTrace{}, lib.NotFound("JUDGMENT_TRACE_NOT_FOUND", "judgment trace not found")
	}
	return item, nil
}

func (s *apiFakeJudgmentTraceStore) ListJudgmentTracesByProject(_ context.Context, projectID string, _ int) ([]domain.JudgmentTrace, error) {
	var items []domain.JudgmentTrace
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type apiFakeHeuristicProposalStore struct {
	items map[string]domain.HeuristicProposal
}

func (s *apiFakeHeuristicProposalStore) CreateHeuristicProposal(_ context.Context, proposal domain.HeuristicProposal) (domain.HeuristicProposal, error) {
	if s.items == nil {
		s.items = map[string]domain.HeuristicProposal{}
	}
	if existing, ok := s.items[proposal.ID]; ok {
		return existing, nil
	}
	s.items[proposal.ID] = proposal
	return proposal, nil
}

func (s *apiFakeHeuristicProposalStore) GetHeuristicProposal(_ context.Context, id string) (domain.HeuristicProposal, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.HeuristicProposal{}, lib.NotFound("HEURISTIC_PROPOSAL_NOT_FOUND", "heuristic proposal not found")
	}
	return item, nil
}

func (s *apiFakeHeuristicProposalStore) ListHeuristicProposalsByProject(_ context.Context, projectID string, state string, _ string, _ int) ([]domain.HeuristicProposal, error) {
	var items []domain.HeuristicProposal
	for _, item := range s.items {
		if item.ProjectID == projectID && (state == "" || item.State == state) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *apiFakeHeuristicProposalStore) UpdateHeuristicProposalState(_ context.Context, id string, state string, reviewNotes string) (domain.HeuristicProposal, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.HeuristicProposal{}, lib.NotFound("HEURISTIC_PROPOSAL_NOT_FOUND", "heuristic proposal not found")
	}
	if item.State != string(contracts.HeuristicStatePending) {
		return domain.HeuristicProposal{}, lib.AppError{
			Code:      "PROPOSAL_ALREADY_RESOLVED",
			Message:   "heuristic proposal has already been resolved",
			Retryable: false,
		}
	}
	item.State = state
	item.ReviewNotes = reviewNotes
	s.items[id] = item
	return item, nil
}

type apiFakeApprovedHeuristicStore struct {
	items map[string]domain.ApprovedHeuristic
}

func (s *apiFakeApprovedHeuristicStore) CreateApprovedHeuristic(_ context.Context, heuristic domain.ApprovedHeuristic) (domain.ApprovedHeuristic, error) {
	if s.items == nil {
		s.items = map[string]domain.ApprovedHeuristic{}
	}
	if existing, ok := s.items[heuristic.ID]; ok {
		return existing, nil
	}
	s.items[heuristic.ID] = heuristic
	return heuristic, nil
}

func (s *apiFakeApprovedHeuristicStore) GetApprovedHeuristic(_ context.Context, id string) (domain.ApprovedHeuristic, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.ApprovedHeuristic{}, lib.NotFound("APPROVED_HEURISTIC_NOT_FOUND", "approved heuristic not found")
	}
	return item, nil
}

func (s *apiFakeApprovedHeuristicStore) ListApprovedHeuristicsByProject(_ context.Context, projectID string, workflow string, artifactType string, _ string, limit int) ([]domain.ApprovedHeuristic, error) {
	var items []domain.ApprovedHeuristic
	for _, item := range s.items {
		if item.ProjectID != projectID || item.State != "approved" {
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

func (s *apiFakeApprovedHeuristicStore) UpdateApprovedHeuristicState(_ context.Context, id string, state string) (domain.ApprovedHeuristic, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.ApprovedHeuristic{}, lib.NotFound("APPROVED_HEURISTIC_NOT_FOUND", "approved heuristic not found")
	}
	item.State = state
	s.items[id] = item
	return item, nil
}

type apiFakeIdempotencyStore struct {
	items map[string]domain.IdempotencyRecord
}

func (s *apiFakeIdempotencyStore) CreateIdempotencyRecord(_ context.Context, record domain.IdempotencyRecord) (domain.IdempotencyRecord, error) {
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

func (s *apiFakeIdempotencyStore) GetIdempotencyRecord(_ context.Context, scopeKind string, scopeProjectID string, key string) (domain.IdempotencyRecord, error) {
	item, ok := s.items[scopeKind+":"+scopeProjectID+":"+key]
	if !ok {
		return domain.IdempotencyRecord{}, lib.NotFound("IDEMPOTENCY_RECORD_NOT_FOUND", "idempotency record not found")
	}
	return item, nil
}

type apiStyleMemoryStores struct {
	traces      *apiFakeJudgmentTraceStore
	proposals   *apiFakeHeuristicProposalStore
	approved    *apiFakeApprovedHeuristicStore
	idempotency *apiFakeIdempotencyStore
}

func TestStyleMemoryRoutesCaptureProposalReviewAndUpdate(t *testing.T) {
	projectID := lib.ProjectID("relay")
	mux, _ := newStyleMemoryTestMux(projectID, nil)

	trace := postStyleMemoryJSON(t, mux, "/v1/judgment-traces", "admin-token", map[string]any{
		"project":         "relay",
		"task_id":         "task-1",
		"agent_id":        "codex",
		"decision":        "Prefer explicit contracts over implicit inference.",
		"rationale":       "Keeps model-to-model handoff deterministic.",
		"language":        "en",
		"idempotency_key": "trace-1",
	})
	traceID := stringField(t, trace, "trace_id")
	if traceID == "" {
		t.Fatalf("expected trace_id in response: %#v", trace)
	}

	proposal := postStyleMemoryJSON(t, mux, "/v1/heuristic-proposals", "admin-token", map[string]any{
		"project":         "relay",
		"origin_trace_id": traceID,
		"workflow":        "design_handoff",
		"artifact_type":   "design_doc",
		"heuristic_key":   "explicit_contracts_over_magic",
		"canonical_text":  "Prefer explicit contracts over magic inference.",
		"proposed_by":     "manual",
		"idempotency_key": "proposal-1",
	})
	proposalID := stringField(t, proposal, "proposal_id")
	if proposalID == "" || stringField(t, proposal, "state") != "pending" {
		t.Fatalf("expected pending proposal response: %#v", proposal)
	}

	review := postStyleMemoryJSON(t, mux, "/v1/heuristic-proposals/review", "admin-token", map[string]any{
		"project":     "relay",
		"proposal_id": proposalID,
		"action":      "approve",
	})
	heuristicID := stringField(t, review, "approved_heuristic_id")
	if heuristicID == "" || stringField(t, review, "state") != "approved" {
		t.Fatalf("expected approved heuristic response: %#v", review)
	}

	update := postStyleMemoryJSON(t, mux, "/v1/approved-heuristics/update", "admin-token", map[string]any{
		"project":      "relay",
		"heuristic_id": heuristicID,
		"action":       "disable",
	})
	if stringField(t, update, "state") != "disabled" {
		t.Fatalf("expected disabled heuristic response: %#v", update)
	}
}

func TestStyleMemoryRoutesRespectProjectScopedAPIKeys(t *testing.T) {
	projectID := lib.ProjectID("relay")
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash("project-token"): {
				ID:        "key_1",
				Name:      "relay project",
				TokenHash: lib.TokenHash("project-token"),
				Scope:     services.APIKeyScopeProject,
				ProjectID: projectID,
			},
		},
	}
	mux, _ := newStyleMemoryTestMux(projectID, keyStore)

	body, err := json.Marshal(map[string]any{
		"project":   "other",
		"task_id":   "task-1",
		"agent_id":  "codex",
		"decision":  "This must not cross project scope.",
		"rationale": "A project-scoped key may only write its bound project.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/judgment-traces", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer project-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStyleMemoryReviewRouteRequiresAdminToken(t *testing.T) {
	projectID := lib.ProjectID("relay")
	keyStore := &fakeAPIKeyStore{
		itemsByHash: map[string]domain.APIKey{
			lib.TokenHash("project-token"): {
				ID:        "key_1",
				Name:      "relay project",
				TokenHash: lib.TokenHash("project-token"),
				Scope:     services.APIKeyScopeProject,
				ProjectID: projectID,
			},
		},
	}
	mux, _ := newStyleMemoryTestMux(projectID, keyStore)
	body, err := json.Marshal(map[string]any{
		"project":     "relay",
		"proposal_id": "hprop_1",
		"action":      "approve",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/heuristic-proposals/review", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer project-token")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func newStyleMemoryTestMux(projectID string, apiKeys *fakeAPIKeyStore) (*http.ServeMux, apiStyleMemoryStores) {
	stores := apiStyleMemoryStores{
		traces:      &apiFakeJudgmentTraceStore{},
		proposals:   &apiFakeHeuristicProposalStore{},
		approved:    &apiFakeApprovedHeuristicStore{},
		idempotency: &apiFakeIdempotencyStore{},
	}
	service := services.New(services.Dependencies{
		Projects: &fakeProjectStore{
			projects: map[string]domain.Project{
				"relay": {ID: projectID, Name: "relay"},
				"other": {ID: lib.ProjectID("other"), Name: "other"},
			},
		},
		Notes:              &fakeNoteStore{},
		Artifacts:          &fakeArtifactStore{},
		Decisions:          &fakeDecisionStore{},
		OpenQuestions:      &fakeOpenQuestionStore{},
		Packets:            &fakePacketStore{},
		APIKeys:            apiKeys,
		JudgmentTraces:     stores.traces,
		HeuristicProposals: stores.proposals,
		ApprovedHeuristics: stores.approved,
		Idempotency:        stores.idempotency,
	})
	return buildMux(Handler{services: service}, config.Config{APIToken: "admin-token"}, app.Runtime{
		Services: service,
		APIKeys:  apiKeys,
	}), stores
}

func postStyleMemoryJSON(t *testing.T, mux *http.ServeMux, path string, token string, payload map[string]any) map[string]any {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return envelope.Data
}

func stringField(t *testing.T, data map[string]any, field string) string {
	t.Helper()
	value, _ := data[field].(string)
	return value
}
