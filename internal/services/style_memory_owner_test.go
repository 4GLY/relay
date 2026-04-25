package services

import (
	"context"
	"testing"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

// TestReviewHeuristicProposalAllowsProjectOwner exercises R1: the legacy
// admin path and the new project-owner path must both succeed, and a
// non-admin / non-owner user must still be rejected.
func TestReviewHeuristicProposalAllowsProjectOwner(t *testing.T) {
	projectID := lib.ProjectID("relay")
	ownerUserID := "usr_owner"

	build := func() (Service, *fakeHeuristicProposalStore, *fakeApprovedHeuristicStore) {
		proposals := &fakeHeuristicProposalStore{
			items: map[string]domain.HeuristicProposal{
				"hprop_1": {
					ID:            "hprop_1",
					ProjectID:     projectID,
					HeuristicKey:  "explicit_contracts_over_magic",
					CanonicalText: "Prefer explicit contracts over magic inference.",
					State:         string(contracts.HeuristicStatePending),
				},
			},
		}
		approved := &fakeApprovedHeuristicStore{}
		svc := New(Dependencies{
			Projects: &fakeProjectStore{projects: map[string]domain.Project{
				"relay": {ID: projectID, Name: "relay", OwnerUserID: ownerUserID},
			}},
			HeuristicProposals: proposals,
			ApprovedHeuristics: approved,
		})
		return svc, proposals, approved
	}

	t.Run("admin still passes", func(t *testing.T) {
		svc, _, approved := build()
		ctx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})
		result, err := svc.ReviewHeuristicProposal(ctx, HeuristicProposalReviewInput{
			Project:    "relay",
			ProposalID: "hprop_1",
			Action:     "approve",
		})
		if err != nil {
			t.Fatalf("admin review error: %v", err)
		}
		if result.ApprovedHeuristicID == "" {
			t.Fatal("expected approved heuristic id")
		}
		if len(approved.items) != 1 {
			t.Fatalf("expected 1 approved heuristic, got %d", len(approved.items))
		}
	})

	t.Run("project owner passes", func(t *testing.T) {
		svc, _, approved := build()
		ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
			UserID: ownerUserID,
			Scope:  APIKeyScopeGlobal,
		})
		result, err := svc.ReviewHeuristicProposal(ctx, HeuristicProposalReviewInput{
			Project:    "relay",
			ProposalID: "hprop_1",
			Action:     "approve",
		})
		if err != nil {
			t.Fatalf("owner review error: %v", err)
		}
		if result.ApprovedHeuristicID == "" {
			t.Fatal("expected approved heuristic id")
		}
		if len(approved.items) != 1 {
			t.Fatalf("expected 1 approved heuristic, got %d", len(approved.items))
		}
	})

	t.Run("other user rejected", func(t *testing.T) {
		svc, _, _ := build()
		ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
			UserID: "usr_intruder",
			Scope:  APIKeyScopeGlobal,
		})
		_, err := svc.ReviewHeuristicProposal(ctx, HeuristicProposalReviewInput{
			Project:    "relay",
			ProposalID: "hprop_1",
			Action:     "approve",
		})
		if err == nil {
			t.Fatal("expected non-owner review to fail")
		}
		appErr, ok := err.(lib.AppError)
		if !ok || appErr.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN, got %#v", err)
		}
	})
}

func TestUpdateApprovedHeuristicAllowsProjectOwner(t *testing.T) {
	projectID := lib.ProjectID("relay")
	ownerUserID := "usr_owner"

	build := func() (Service, *fakeApprovedHeuristicStore) {
		approved := &fakeApprovedHeuristicStore{
			items: map[string]domain.ApprovedHeuristic{
				"heur_1": {
					ID:            "heur_1",
					ProjectID:     projectID,
					HeuristicKey:  "explicit_contracts_over_magic",
					CanonicalText: "Prefer explicit contracts over magic inference.",
					State:         string(contracts.HeuristicStateApproved),
				},
			},
		}
		svc := New(Dependencies{
			Projects: &fakeProjectStore{projects: map[string]domain.Project{
				"relay": {ID: projectID, Name: "relay", OwnerUserID: ownerUserID},
			}},
			ApprovedHeuristics: approved,
		})
		return svc, approved
	}

	t.Run("admin still passes", func(t *testing.T) {
		svc, approved := build()
		ctx := ContextWithAuthInfo(context.Background(), AuthInfo{IsAdmin: true, Scope: APIKeyScopeGlobal})
		result, err := svc.UpdateApprovedHeuristic(ctx, ApprovedHeuristicUpdateInput{
			Project:     "relay",
			HeuristicID: "heur_1",
			Action:      "disable",
		})
		if err != nil {
			t.Fatalf("admin update error: %v", err)
		}
		if result.State != string(contracts.HeuristicStateDisabled) {
			t.Fatalf("expected disabled state, got %q", result.State)
		}
		if approved.items["heur_1"].State != string(contracts.HeuristicStateDisabled) {
			t.Fatalf("expected stored state disabled, got %q", approved.items["heur_1"].State)
		}
	})

	t.Run("project owner passes", func(t *testing.T) {
		svc, approved := build()
		ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
			UserID: ownerUserID,
			Scope:  APIKeyScopeGlobal,
		})
		result, err := svc.UpdateApprovedHeuristic(ctx, ApprovedHeuristicUpdateInput{
			Project:     "relay",
			HeuristicID: "heur_1",
			Action:      "disable",
		})
		if err != nil {
			t.Fatalf("owner update error: %v", err)
		}
		if result.State != string(contracts.HeuristicStateDisabled) {
			t.Fatalf("expected disabled state, got %q", result.State)
		}
		if approved.items["heur_1"].State != string(contracts.HeuristicStateDisabled) {
			t.Fatalf("expected stored state disabled, got %q", approved.items["heur_1"].State)
		}
	})

	t.Run("other user rejected", func(t *testing.T) {
		svc, _ := build()
		ctx := ContextWithAuthInfo(context.Background(), AuthInfo{
			UserID: "usr_intruder",
			Scope:  APIKeyScopeGlobal,
		})
		_, err := svc.UpdateApprovedHeuristic(ctx, ApprovedHeuristicUpdateInput{
			Project:     "relay",
			HeuristicID: "heur_1",
			Action:      "disable",
		})
		if err == nil {
			t.Fatal("expected non-owner update to fail")
		}
		appErr, ok := err.(lib.AppError)
		if !ok || appErr.Code != "FORBIDDEN" {
			t.Fatalf("expected FORBIDDEN, got %#v", err)
		}
	})
}
