package services

import (
	"context"
	"sort"
	"strings"

	"relay/internal/domain"
	"relay/internal/lib"
)

type packetSnapshotExplorerStore interface {
	CountPacketSnapshotsByProject(ctx context.Context, projectID string) (int, error)
	LatestAnyPacketSnapshotByProject(ctx context.Context, projectID string) (domain.PacketSnapshot, error)
}

func (s Service) ProjectExplorer(ctx context.Context, input ProjectExplorerInput) (ProjectExplorerResult, error) {
	if strings.TrimSpace(input.ProjectID) == "" {
		return ProjectExplorerResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project_id")
	}
	if s.deps.Projects == nil {
		return ProjectExplorerResult{}, lib.Misconfigured("project store is required")
	}
	project, err := s.deps.Projects.GetByID(ctx, input.ProjectID)
	if err != nil {
		return ProjectExplorerResult{}, err
	}
	if err := s.requireProjectRead(ctx, project.ID); err != nil {
		return ProjectExplorerResult{}, err
	}

	counts, err := s.projectExplorerCounts(ctx, project.ID)
	if err != nil {
		return ProjectExplorerResult{}, err
	}
	pending, err := s.listExplorerProposals(ctx, project.ID, "pending", 1)
	if err != nil {
		return ProjectExplorerResult{}, err
	}
	traces, err := s.listExplorerTraces(ctx, project.ID, 5)
	if err != nil {
		return ProjectExplorerResult{}, err
	}
	approved, err := s.listExplorerApproved(ctx, project.ID, 5)
	if err != nil {
		return ProjectExplorerResult{}, err
	}
	latestSnapshot, err := s.latestExplorerSnapshot(ctx, project.ID)
	if err != nil {
		return ProjectExplorerResult{}, err
	}

	style := ProjectExplorerStyleMemory{}
	if len(pending) > 0 {
		style.NextProposalID = pending[0].ID
		style.NextProposalText = pending[0].CanonicalText
	}

	return ProjectExplorerResult{
		Project: ProjectExplorerProject{
			ProjectID: project.ID,
			Name:      project.Name,
			Status:    project.Status,
		},
		Counts:         counts,
		LatestSnapshot: latestSnapshot,
		StyleMemory:    style,
		RecentActivity: recentExplorerActivity(traces, approved),
	}, nil
}

func (s Service) ListJudgmentTraces(ctx context.Context, input ListJudgmentTracesInput) (ListJudgmentTracesResult, error) {
	if strings.TrimSpace(input.ProjectID) == "" {
		return ListJudgmentTracesResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project_id")
	}
	if s.deps.Projects == nil {
		return ListJudgmentTracesResult{}, lib.Misconfigured("project store is required")
	}
	project, err := s.deps.Projects.GetByID(ctx, input.ProjectID)
	if err != nil {
		return ListJudgmentTracesResult{}, err
	}
	if err := s.requireProjectRead(ctx, project.ID); err != nil {
		return ListJudgmentTracesResult{}, err
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	traces, err := s.listExplorerTraces(ctx, project.ID, limit)
	if err != nil {
		return ListJudgmentTracesResult{}, err
	}
	items := make([]JudgmentTraceSummary, 0, len(traces))
	for _, trace := range traces {
		items = append(items, JudgmentTraceSummary{
			TraceID:      trace.ID,
			ProjectID:    trace.ProjectID,
			TaskID:       trace.TaskID,
			AgentID:      trace.AgentID,
			Workflow:     trace.Workflow,
			ArtifactType: trace.ArtifactType,
			Decision:     trace.Decision,
			Rationale:    trace.Rationale,
			SourceRefs:   trace.SourceRefs,
			CreatedAt:    trace.CreatedAt,
		})
	}
	return ListJudgmentTracesResult{Items: items}, nil
}

func (s Service) requireProjectRead(ctx context.Context, projectID string) error {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok {
		return lib.Forbidden("FORBIDDEN", "authorization is required")
	}
	if auth.IsAdmin {
		return nil
	}
	if auth.KeyID != "" && NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeGlobal {
		return nil
	}
	if NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeProject {
		if auth.ProjectID == projectID {
			return nil
		}
		return lib.Forbidden("FORBIDDEN", "api key is not authorized for this project")
	}
	if auth.UserID == "" {
		return lib.Forbidden("FORBIDDEN", "project owner authorization is required")
	}
	project, err := s.deps.Projects.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != "" && project.OwnerUserID == auth.UserID {
		return nil
	}
	return lib.Forbidden("FORBIDDEN", "project owner authorization is required")
}

func (s Service) projectExplorerCounts(ctx context.Context, projectID string) (ProjectExplorerCounts, error) {
	counts := ProjectExplorerCounts{}
	if s.deps.Notes != nil {
		n, err := s.deps.Notes.CountByProject(ctx, projectID)
		if err != nil {
			return counts, err
		}
		counts.Notes = n
	}
	if s.deps.Artifacts != nil {
		n, err := s.deps.Artifacts.CountByProject(ctx, projectID)
		if err != nil {
			return counts, err
		}
		counts.Artifacts = n
	}
	if s.deps.Decisions != nil {
		n, err := s.deps.Decisions.CountByProject(ctx, projectID)
		if err != nil {
			return counts, err
		}
		counts.Decisions = n
	}
	if s.deps.OpenQuestions != nil {
		n, err := s.deps.OpenQuestions.CountByProject(ctx, projectID)
		if err != nil {
			return counts, err
		}
		counts.OpenQuestions = n
	}

	traces, err := s.listExplorerTraces(ctx, projectID, 101)
	if err != nil {
		return counts, err
	}
	counts.JudgmentTraces = len(traces)
	counts.PendingProposals, err = s.countExplorerProposals(ctx, projectID, "pending")
	if err != nil {
		return counts, err
	}
	counts.RejectedProposals, err = s.countExplorerProposals(ctx, projectID, "rejected")
	if err != nil {
		return counts, err
	}
	approved, err := s.listExplorerApproved(ctx, projectID, 101)
	if err != nil {
		return counts, err
	}
	counts.ApprovedHeuristics = len(approved)
	if store, ok := s.deps.PacketSnapshots.(packetSnapshotExplorerStore); ok {
		n, err := store.CountPacketSnapshotsByProject(ctx, projectID)
		if err != nil {
			return counts, err
		}
		counts.PacketSnapshots = n
	}
	return counts, nil
}

func (s Service) countExplorerProposals(ctx context.Context, projectID string, state string) (int, error) {
	proposals, err := s.listExplorerProposals(ctx, projectID, state, 101)
	if err != nil {
		return 0, err
	}
	return len(proposals), nil
}

func (s Service) listExplorerTraces(ctx context.Context, projectID string, limit int) ([]domain.JudgmentTrace, error) {
	if s.deps.JudgmentTraces == nil {
		return nil, nil
	}
	return s.deps.JudgmentTraces.ListJudgmentTracesByProject(ctx, projectID, limit)
}

func (s Service) listExplorerProposals(ctx context.Context, projectID string, state string, limit int) ([]domain.HeuristicProposal, error) {
	if s.deps.HeuristicProposals == nil {
		return nil, nil
	}
	return s.deps.HeuristicProposals.ListHeuristicProposalsByProject(ctx, projectID, state, "", limit)
}

func (s Service) listExplorerApproved(ctx context.Context, projectID string, limit int) ([]domain.ApprovedHeuristic, error) {
	if s.deps.ApprovedHeuristics == nil {
		return nil, nil
	}
	return s.deps.ApprovedHeuristics.ListApprovedHeuristicsByProject(ctx, projectID, "", "", "", limit)
}

func (s Service) latestExplorerSnapshot(ctx context.Context, projectID string) (*ProjectExplorerSnapshot, error) {
	store, ok := s.deps.PacketSnapshots.(packetSnapshotExplorerStore)
	if !ok {
		return nil, nil
	}
	snapshot, err := store.LatestAnyPacketSnapshotByProject(ctx, projectID)
	if err != nil {
		if isNotFound(err, "PACKET_SNAPSHOT_NOT_FOUND") {
			return nil, nil
		}
		return nil, err
	}
	return &ProjectExplorerSnapshot{
		SnapshotID:     snapshot.ID,
		PacketKind:     snapshot.PacketKind,
		Target:         snapshot.Target,
		TaskSummary:    snapshot.TaskSummary,
		CreatedAt:      snapshot.CreatedAt,
		PublicReadable: snapshot.PublicReadable,
		PublicToken:    publicSnapshotToken(snapshot),
	}, nil
}

func publicSnapshotToken(snapshot domain.PacketSnapshot) string {
	if !snapshot.PublicReadable {
		return ""
	}
	return snapshot.PublicToken
}

func recentExplorerActivity(traces []domain.JudgmentTrace, approved []domain.ApprovedHeuristic) []ProjectExplorerActivity {
	items := make([]ProjectExplorerActivity, 0, len(traces)+len(approved))
	for _, trace := range traces {
		items = append(items, ProjectExplorerActivity{
			Kind:      "judgment_trace",
			ID:        trace.ID,
			Title:     firstNonEmpty(trace.Decision, trace.Rationale, trace.Workflow),
			CreatedAt: trace.CreatedAt,
		})
	}
	for _, heuristic := range approved {
		items = append(items, ProjectExplorerActivity{
			Kind:      "approved_heuristic",
			ID:        heuristic.ID,
			Title:     firstNonEmpty(heuristic.CanonicalText, heuristic.HeuristicKey),
			CreatedAt: heuristic.CreatedAt,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		a := items[i].CreatedAt
		b := items[j].CreatedAt
		if a.IsZero() && b.IsZero() {
			return items[i].ID < items[j].ID
		}
		if a.IsZero() {
			return false
		}
		if b.IsZero() {
			return true
		}
		return a.After(b)
	})
	if len(items) > 5 {
		items = items[:5]
	}
	return items
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "Untitled"
}
