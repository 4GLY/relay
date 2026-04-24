package repositories

import (
	"context"
	"time"

	"relay/internal/domain"
)

type ProjectStore interface {
	EnsureProject(ctx context.Context, project domain.Project) (domain.Project, error)
	GetByID(ctx context.Context, id string) (domain.Project, error)
	GetByName(ctx context.Context, name string) (domain.Project, error)
}

type NoteStore interface {
	CreateNote(ctx context.Context, note domain.Note) (domain.Note, error)
	CountByProject(ctx context.Context, projectID string) (int, error)
	ListByProject(ctx context.Context, projectID string) ([]domain.Note, error)
}

type ArtifactStore interface {
	CreateArtifact(ctx context.Context, artifact domain.Artifact) (domain.Artifact, error)
	CountByProject(ctx context.Context, projectID string) (int, error)
	ListByProject(ctx context.Context, projectID string) ([]domain.Artifact, error)
}

type DecisionStore interface {
	CreateDecision(ctx context.Context, decision domain.Decision) (domain.Decision, error)
	CountByProject(ctx context.Context, projectID string) (int, error)
	ListByProject(ctx context.Context, projectID string) ([]domain.Decision, error)
}

type OpenQuestionStore interface {
	CreateOpenQuestion(ctx context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error)
	CountByProject(ctx context.Context, projectID string) (int, error)
	ListByProject(ctx context.Context, projectID string) ([]domain.OpenQuestion, error)
}

type PacketStore interface {
	CreatePacket(ctx context.Context, packet domain.Packet) (domain.Packet, error)
	LatestByProject(ctx context.Context, projectID string) (domain.Packet, error)
}

type APIKeyStore interface {
	CreateAPIKey(ctx context.Context, key domain.APIKey) (domain.APIKey, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (domain.APIKey, error)
	ListAPIKeys(ctx context.Context) ([]domain.APIKey, error)
	RevokeAPIKey(ctx context.Context, keyID string) (domain.APIKey, error)
}

type JudgmentTraceStore interface {
	CreateJudgmentTrace(ctx context.Context, trace domain.JudgmentTrace) (domain.JudgmentTrace, error)
	GetJudgmentTrace(ctx context.Context, id string) (domain.JudgmentTrace, error)
	ListJudgmentTracesByProject(ctx context.Context, projectID string, limit int) ([]domain.JudgmentTrace, error)
}

type HeuristicProposalStore interface {
	CreateHeuristicProposal(ctx context.Context, proposal domain.HeuristicProposal) (domain.HeuristicProposal, error)
	GetHeuristicProposal(ctx context.Context, id string) (domain.HeuristicProposal, error)
	ListHeuristicProposalsByProject(ctx context.Context, projectID string, state string, limit int) ([]domain.HeuristicProposal, error)
	UpdateHeuristicProposalState(ctx context.Context, id string, state string, reviewNotes string) (domain.HeuristicProposal, error)
}

type ApprovedHeuristicStore interface {
	CreateApprovedHeuristic(ctx context.Context, heuristic domain.ApprovedHeuristic) (domain.ApprovedHeuristic, error)
	GetApprovedHeuristic(ctx context.Context, id string) (domain.ApprovedHeuristic, error)
	ListApprovedHeuristicsByProject(ctx context.Context, projectID string, workflow string, artifactType string, limit int) ([]domain.ApprovedHeuristic, error)
	UpdateApprovedHeuristicState(ctx context.Context, id string, state string) (domain.ApprovedHeuristic, error)
}

type PacketSnapshotStore interface {
	CreatePacketSnapshot(ctx context.Context, snapshot domain.PacketSnapshot) (domain.PacketSnapshot, error)
	GetPacketSnapshot(ctx context.Context, id string) (domain.PacketSnapshot, error)
	LatestPacketSnapshotByProject(ctx context.Context, projectID string, packetKind string, target string) (domain.PacketSnapshot, error)
}

type IdempotencyStore interface {
	CreateIdempotencyRecord(ctx context.Context, record domain.IdempotencyRecord) (domain.IdempotencyRecord, error)
	GetIdempotencyRecord(ctx context.Context, scopeKind string, scopeProjectID string, key string) (domain.IdempotencyRecord, error)
}

type CuratorJobStore interface {
	EnqueueCuratorJob(ctx context.Context, job domain.CuratorJob) (domain.CuratorJob, error)
	ClaimCuratorJobs(ctx context.Context, owner string, limit int, leaseUntil time.Time) ([]domain.CuratorJob, error)
	CompleteCuratorJob(ctx context.Context, id string) (domain.CuratorJob, error)
	FailCuratorJob(ctx context.Context, id string, retryAt time.Time, lastError string) (domain.CuratorJob, error)
	MarkCuratorJobFailed(ctx context.Context, id string, lastError string) (domain.CuratorJob, error)
}
