package repositories

import (
	"context"

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
