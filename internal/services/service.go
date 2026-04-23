package services

import "relay/internal/storage/repositories"

type Dependencies struct {
	Projects           repositories.ProjectStore
	Notes              repositories.NoteStore
	Artifacts          repositories.ArtifactStore
	Decisions          repositories.DecisionStore
	OpenQuestions      repositories.OpenQuestionStore
	Packets            repositories.PacketStore
	APIKeys            repositories.APIKeyStore
	JudgmentTraces     repositories.JudgmentTraceStore
	HeuristicProposals repositories.HeuristicProposalStore
	ApprovedHeuristics repositories.ApprovedHeuristicStore
	PacketSnapshots    repositories.PacketSnapshotStore
	Idempotency        repositories.IdempotencyStore
	CuratorJobs        repositories.CuratorJobStore
}

type Service struct {
	deps Dependencies
}

func New(deps Dependencies) Service {
	return Service{deps: deps}
}
