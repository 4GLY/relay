package services

import (
	"relay/internal/lib/crypto"
	"relay/internal/storage/repositories"
)

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
	OGImages           OGImageWriter
	CacheInvalidator   CacheInvalidator
	PublicBaseURL      string
	Users              repositories.UserStore
	OAuthIdentities    repositories.OAuthIdentityStore
	UserSessions       repositories.UserSessionStore
	OAuthStates        repositories.OAuthStateStore
	Onboarding         repositories.OnboardingStore
}

type Service struct {
	deps             Dependencies
	keks             map[crypto.KEKVersion][]byte
	activeKEKVersion crypto.KEKVersion
}

func New(deps Dependencies) Service {
	return Service{deps: deps}
}

// NewWithKEKs builds a Service that can encrypt/decrypt envelope-sealed
// secrets (Anthropic key, future PII). Used by app.NewRuntime at boot. Tests
// that don't exercise the onboarding path can keep using New(deps).
func NewWithKEKs(deps Dependencies, keks map[crypto.KEKVersion][]byte, active crypto.KEKVersion) Service {
	return Service{deps: deps, keks: keks, activeKEKVersion: active}
}
