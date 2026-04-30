package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
)

func TestProjectExplorerSummarizesOwnedProject(t *testing.T) {
	projectID := "proj_owner"
	userID := "user_owner"
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"Personal": {ID: projectID, Name: "Personal", Status: "active", OwnerUserID: userID},
		}},
		Notes:         &fakeNoteStore{items: []domain.Note{{ID: "note_1", ProjectID: projectID}}},
		Artifacts:     &fakeArtifactStore{items: []domain.Artifact{{ID: "art_1", ProjectID: projectID}}},
		Decisions:     &fakeDecisionStore{items: []domain.Decision{{ID: "dec_1", ProjectID: projectID}}},
		OpenQuestions: &fakeOpenQuestionStore{items: []domain.OpenQuestion{{ID: "oq_1", ProjectID: projectID}}},
		JudgmentTraces: &fakeJudgmentTraceStore{items: map[string]domain.JudgmentTrace{
			"trace_1": {
				ID:        "trace_1",
				ProjectID: projectID,
				Decision:  "Prefer specific recovery actions.",
				CreatedAt: time.Date(2026, 4, 29, 2, 0, 0, 0, time.UTC),
			},
		}},
		HeuristicProposals: &fakeHeuristicProposalStore{items: map[string]domain.HeuristicProposal{
			"prop_pending": {
				ID:            "prop_pending",
				ProjectID:     projectID,
				HeuristicKey:  "specific_recovery",
				CanonicalText: "Show a specific recovery action.",
				State:         "pending",
				CreatedAt:     time.Date(2026, 4, 29, 2, 1, 0, 0, time.UTC),
			},
			"prop_rejected": {
				ID:            "prop_rejected",
				ProjectID:     projectID,
				HeuristicKey:  "old",
				CanonicalText: "Old proposal.",
				State:         "rejected",
			},
		}},
		ApprovedHeuristics: &fakeApprovedHeuristicStore{items: map[string]domain.ApprovedHeuristic{
			"heur_1": {
				ID:            "heur_1",
				ProjectID:     projectID,
				HeuristicKey:  "specific_recovery",
				CanonicalText: "Show a specific recovery action.",
				State:         "approved",
				CreatedAt:     time.Date(2026, 4, 29, 2, 2, 0, 0, time.UTC),
			},
		}},
		PacketSnapshots: &fakePacketSnapshotStore{items: map[string]domain.PacketSnapshot{
			"psnap_1": {
				ID:             "psnap_1",
				ProjectID:      projectID,
				PacketKind:     "handoff",
				Target:         "design_doc",
				TaskSummary:    "Prepare design handoff",
				PublicReadable: true,
				PublicToken:    "psnap_public_token",
				CreatedAt:      time.Date(2026, 4, 29, 2, 3, 0, 0, time.UTC),
			},
		}},
	})
	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{UserID: userID, Scope: APIKeyScopeGlobal})

	result, err := service.ProjectExplorer(ctx, ProjectExplorerInput{ProjectID: projectID})
	if err != nil {
		t.Fatalf("ProjectExplorer returned error: %v", err)
	}
	if result.Project.ProjectID != projectID || result.Project.Name != "Personal" {
		t.Fatalf("unexpected project summary: %#v", result.Project)
	}
	if result.Counts.Notes != 1 || result.Counts.JudgmentTraces != 1 || result.Counts.PendingProposals != 1 || result.Counts.ApprovedHeuristics != 1 || result.Counts.RejectedProposals != 1 || result.Counts.PacketSnapshots != 1 {
		t.Fatalf("unexpected counts: %#v", result.Counts)
	}
	if result.LatestSnapshot == nil || result.LatestSnapshot.SnapshotID != "psnap_1" || !result.LatestSnapshot.PublicReadable {
		t.Fatalf("unexpected latest snapshot: %#v", result.LatestSnapshot)
	}
	if result.LatestSnapshot.PublicToken != "psnap_public_token" {
		t.Fatalf("expected public snapshot token, got %#v", result.LatestSnapshot)
	}
	if result.StyleMemory.NextProposalID != "prop_pending" {
		t.Fatalf("unexpected style memory preview: %#v", result.StyleMemory)
	}
	if len(result.RecentActivity) == 0 || result.RecentActivity[0].Kind != "approved_heuristic" {
		t.Fatalf("expected recent approved heuristic first: %#v", result.RecentActivity)
	}
}

func TestProjectExplorerRejectsNonOwnerSession(t *testing.T) {
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"Personal": {ID: "proj_owner", Name: "Personal", Status: "active", OwnerUserID: "user_owner"},
		}},
	})
	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{UserID: "user_other", Scope: APIKeyScopeGlobal})

	_, err := service.ProjectExplorer(ctx, ProjectExplorerInput{ProjectID: "proj_owner"})
	if err == nil {
		t.Fatal("expected non-owner session to be rejected")
	}
	var appErr lib.AppError
	if !errors.As(err, &appErr) || appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %v", err)
	}
}

func TestListJudgmentTracesUsesProjectReadAuthorization(t *testing.T) {
	projectID := "proj_owner"
	service := New(Dependencies{
		Projects: &fakeProjectStore{projects: map[string]domain.Project{
			"Personal": {ID: projectID, Name: "Personal", Status: "active", OwnerUserID: "user_owner"},
		}},
		JudgmentTraces: &fakeJudgmentTraceStore{items: map[string]domain.JudgmentTrace{
			"trace_1": {ID: "trace_1", ProjectID: projectID, Decision: "Keep recovery actions specific."},
		}},
	})
	ctx := ContextWithAuthInfo(context.Background(), AuthInfo{UserID: "user_owner", Scope: APIKeyScopeGlobal})

	result, err := service.ListJudgmentTraces(ctx, ListJudgmentTracesInput{ProjectID: projectID})
	if err != nil {
		t.Fatalf("ListJudgmentTraces returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].TraceID != "trace_1" {
		t.Fatalf("unexpected traces: %#v", result.Items)
	}
}
