package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/services"
)

// styleMemorySessionFixture wires session-cookie auth + style-memory routes
// against in-memory fakes so the tests in this file can exercise the
// requireSessionOrAdmin middleware end-to-end.
type styleMemorySessionFixture struct {
	mux        *http.ServeMux
	stores     apiStyleMemoryStores
	users      *apiFakeUserStore
	sessions   *apiFakeUserSessionStore
	projects   *fakeProjectStore
	cookie     *http.Cookie
	otherToken string
	adminToken string
	relayID    string
	otherID    string
	ownerID    string
	otherOwner string
}

func newStyleMemorySessionFixture(t *testing.T) *styleMemorySessionFixture {
	t.Helper()
	relayID := lib.ProjectID("relay")
	otherID := lib.ProjectID("other")
	ownerID := "usr_owner_relay"
	otherOwner := "usr_owner_other"

	users := newAuthFakeUserStore()
	users.items[ownerID] = domain.User{ID: ownerID, Email: "owner@relay.test", DisplayName: "owner"}
	users.items[otherOwner] = domain.User{ID: otherOwner, Email: "other@relay.test", DisplayName: "other"}

	sessions := newAuthFakeUserSessionStore()
	ownerToken, err := lib.NewSecretToken("rsess")
	if err != nil {
		t.Fatalf("create owner token: %v", err)
	}
	if _, err := sessions.CreateUserSession(context.Background(), domain.UserSession{
		ID:        "usess_owner",
		UserID:    ownerID,
		TokenHash: lib.TokenHash(ownerToken),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	otherToken, err := lib.NewSecretToken("rsess")
	if err != nil {
		t.Fatalf("create other token: %v", err)
	}
	if _, err := sessions.CreateUserSession(context.Background(), domain.UserSession{
		ID:        "usess_other",
		UserID:    otherOwner,
		TokenHash: lib.TokenHash(otherToken),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	stores := apiStyleMemoryStores{
		traces:      &apiFakeJudgmentTraceStore{},
		proposals:   &apiFakeHeuristicProposalStore{},
		approved:    &apiFakeApprovedHeuristicStore{},
		idempotency: &apiFakeIdempotencyStore{},
	}
	projectStore := &fakeProjectStore{
		projects: map[string]domain.Project{
			"relay": {ID: relayID, Name: "relay", OwnerUserID: ownerID},
			"other": {ID: otherID, Name: "other", OwnerUserID: otherOwner},
		},
	}
	svc := services.New(services.Dependencies{
		Projects:           projectStore,
		Notes:              &fakeNoteStore{},
		Artifacts:          &fakeArtifactStore{},
		Decisions:          &fakeDecisionStore{},
		OpenQuestions:      &fakeOpenQuestionStore{},
		Packets:            &fakePacketStore{},
		JudgmentTraces:     stores.traces,
		HeuristicProposals: stores.proposals,
		ApprovedHeuristics: stores.approved,
		Idempotency:        stores.idempotency,
		Users:              users,
		UserSessions:       sessions,
	})

	const adminToken = "admin-token"
	mux := buildMux(Handler{services: svc}, config.Config{APIToken: adminToken}, app.Runtime{Services: svc})

	return &styleMemorySessionFixture{
		mux:        mux,
		stores:     stores,
		users:      users,
		sessions:   sessions,
		projects:   projectStore,
		cookie:     &http.Cookie{Name: sessionCookieName, Value: ownerToken},
		otherToken: otherToken,
		adminToken: adminToken,
		relayID:    relayID,
		otherID:    otherID,
		ownerID:    ownerID,
		otherOwner: otherOwner,
	}
}

// T1: cookie-session owner GET on both list routes returns 200 with envelope.
func TestStyleMemoryListsAllowProjectOwnerSession(t *testing.T) {
	f := newStyleMemorySessionFixture(t)

	f.stores.proposals.items = map[string]domain.HeuristicProposal{
		"hprop_1": {
			ID:            "hprop_1",
			ProjectID:     f.relayID,
			HeuristicKey:  "explicit_contracts_over_magic",
			CanonicalText: "Prefer explicit contracts over magic inference.",
			State:         string(contracts.HeuristicStatePending),
			CreatedAt:     time.Now(),
		},
	}
	f.stores.approved.items = map[string]domain.ApprovedHeuristic{
		"heur_1": {
			ID:            "heur_1",
			ProjectID:     f.relayID,
			HeuristicKey:  "explicit_contracts_over_magic",
			CanonicalText: "Prefer explicit contracts over magic inference.",
			State:         string(contracts.HeuristicStateApproved),
			CreatedAt:     time.Now(),
		},
	}

	for _, path := range []string{
		"/v1/heuristic-proposals?project_id=" + f.relayID + "&state=pending",
		"/v1/approved-heuristics?project_id=" + f.relayID,
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(f.cookie)
		rec := httptest.NewRecorder()
		f.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("path %s: expected 200, got %d body=%s", path, rec.Code, rec.Body.String())
		}
		var envelope struct {
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("path %s: decode envelope: %v", path, err)
		}
		if len(envelope.Data.Items) != 1 {
			t.Fatalf("path %s: expected 1 item, got %d body=%s", path, len(envelope.Data.Items), rec.Body.String())
		}
	}
}

// T2: non-owner cookie session attempting to read another project's list → 403.
func TestStyleMemoryListsRejectCrossProjectSession(t *testing.T) {
	f := newStyleMemorySessionFixture(t)

	for _, path := range []string{
		"/v1/heuristic-proposals?project_id=" + f.otherID,
		"/v1/approved-heuristics?project_id=" + f.otherID,
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(f.cookie)
		rec := httptest.NewRecorder()
		f.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("path %s: expected 403, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

// T2 (auth boundary): unauthenticated GET → 401.
func TestStyleMemoryListsRejectAnonymous(t *testing.T) {
	f := newStyleMemorySessionFixture(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/heuristic-proposals?project_id="+f.relayID, nil)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// Admin bearer keeps reaching the new GET routes (R1 regression for the
// auth-wrapper swap on review/update plus the new lists).
func TestStyleMemoryRoutesAcceptAdminBearer(t *testing.T) {
	f := newStyleMemorySessionFixture(t)
	f.stores.proposals.items = map[string]domain.HeuristicProposal{
		"hprop_admin": {
			ID:            "hprop_admin",
			ProjectID:     f.relayID,
			HeuristicKey:  "x",
			CanonicalText: "x",
			State:         string(contracts.HeuristicStatePending),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/heuristic-proposals?project_id="+f.relayID, nil)
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin GET expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	body, _ := json.Marshal(map[string]any{
		"project_id":  f.relayID,
		"proposal_id": "hprop_admin",
		"action":      "approve",
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/heuristic-proposals/review", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	rec = httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin review expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// T3: two concurrent approves on the same proposal — first 200, second 409 with
// PROPOSAL_ALREADY_RESOLVED.
func TestReviewHeuristicProposalCASConflict(t *testing.T) {
	f := newStyleMemorySessionFixture(t)
	f.stores.proposals.items = map[string]domain.HeuristicProposal{
		"hprop_race": {
			ID:            "hprop_race",
			ProjectID:     f.relayID,
			HeuristicKey:  "k",
			CanonicalText: "c",
			State:         string(contracts.HeuristicStatePending),
		},
	}

	postReview := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{
			"project_id":  f.relayID,
			"proposal_id": "hprop_race",
			"action":      "approve",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/heuristic-proposals/review", bytes.NewReader(body))
		req.AddCookie(f.cookie)
		rec := httptest.NewRecorder()
		f.mux.ServeHTTP(rec, req)
		return rec
	}

	first := postReview()
	if first.Code != http.StatusOK {
		t.Fatalf("first review expected 200, got %d body=%s", first.Code, first.Body.String())
	}

	second := postReview()
	if second.Code != http.StatusConflict {
		t.Fatalf("second review expected 409, got %d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Body.String(), "PROPOSAL_ALREADY_RESOLVED") {
		t.Fatalf("expected PROPOSAL_ALREADY_RESOLVED in body, got %s", second.Body.String())
	}
}

// T4: approved-list filter regression — empty workflow filter must include
// heuristics with non-empty workflow values.
func TestApprovedHeuristicsListIncludesAllWhenFilterEmpty(t *testing.T) {
	f := newStyleMemorySessionFixture(t)
	now := time.Now()
	f.stores.approved.items = map[string]domain.ApprovedHeuristic{
		"heur_design": {
			ID:            "heur_design",
			ProjectID:     f.relayID,
			Workflow:      "design_handoff",
			HeuristicKey:  "design",
			CanonicalText: "design rule",
			State:         string(contracts.HeuristicStateApproved),
			CreatedAt:     now,
		},
		"heur_blank": {
			ID:            "heur_blank",
			ProjectID:     f.relayID,
			Workflow:      "",
			HeuristicKey:  "blank",
			CanonicalText: "blank rule",
			State:         string(contracts.HeuristicStateApproved),
			CreatedAt:     now.Add(-time.Second),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/approved-heuristics?project_id="+f.relayID, nil)
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(envelope.Data.Items) != 2 {
		t.Fatalf("expected both heuristics returned with empty workflow filter, got %d body=%s", len(envelope.Data.Items), rec.Body.String())
	}
}

// T5: strict decoder rejects payloads with unknown reject_reason field. This
// asserts the decision (locked) that reject reason is serialized into
// review_notes by the web client, NOT shipped as a typed payload field.
func TestReviewProposalRejectsUnknownRejectReasonField(t *testing.T) {
	f := newStyleMemorySessionFixture(t)
	f.stores.proposals.items = map[string]domain.HeuristicProposal{
		"hprop_unknown": {
			ID:            "hprop_unknown",
			ProjectID:     f.relayID,
			HeuristicKey:  "k",
			CanonicalText: "c",
			State:         string(contracts.HeuristicStatePending),
		},
	}

	body, _ := json.Marshal(map[string]any{
		"project_id":    f.relayID,
		"proposal_id":   "hprop_unknown",
		"action":        "reject",
		"reject_reason": "duplicate",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/heuristic-proposals/review", bytes.NewReader(body))
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "UNKNOWN_JSON_FIELD") {
		t.Fatalf("expected UNKNOWN_JSON_FIELD code in body, got %s", rec.Body.String())
	}
}

// Sanity check that two parallel goroutines hitting the review endpoint also
// produce exactly one 200 and one 409 — exercising the CAS path under load.
func TestReviewHeuristicProposalCASParallelGoroutines(t *testing.T) {
	f := newStyleMemorySessionFixture(t)
	f.stores.proposals.items = map[string]domain.HeuristicProposal{
		"hprop_parallel": {
			ID:            "hprop_parallel",
			ProjectID:     f.relayID,
			HeuristicKey:  "k",
			CanonicalText: "c",
			State:         string(contracts.HeuristicStatePending),
		},
	}

	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			body, _ := json.Marshal(map[string]any{
				"project_id":  f.relayID,
				"proposal_id": "hprop_parallel",
				"action":      "approve",
			})
			req := httptest.NewRequest(http.MethodPost, "/v1/heuristic-proposals/review", bytes.NewReader(body))
			req.AddCookie(f.cookie)
			rec := httptest.NewRecorder()
			f.mux.ServeHTTP(rec, req)
			codes[idx] = rec.Code
		}(i)
	}
	wg.Wait()

	successCount, conflictCount := 0, 0
	for _, code := range codes {
		switch code {
		case http.StatusOK:
			successCount++
		case http.StatusConflict:
			conflictCount++
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("expected exactly one 200 and one 409, got %v", codes)
	}
}
