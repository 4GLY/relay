package app

import (
	"context"

	"relay/internal/config"
	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/crypto"
	"relay/internal/services"
	"relay/internal/storage"
	"relay/internal/storage/postgres"
	"relay/internal/storage/repositories"
)

type Runtime struct {
	Services services.Service
	APIKeys  repositories.APIKeyStore
}

func NewRuntime(ctx context.Context, cfg config.Config) (Runtime, error) {
	if cfg.DatabaseURL == "" {
		return Runtime{}, lib.Misconfigured("RELAY_DATABASE_URL is required")
	}

	db, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return Runtime{}, err
	}
	if err := storage.ApplyMigrations(ctx, db); err != nil {
		db.Close()
		return Runtime{}, err
	}

	stores := postgres.New(db)
	ogWriter, err := services.NewFilesystemOGImageWriter(cfg.OGImageDir)
	if err != nil {
		db.Close()
		return Runtime{}, err
	}
	keks, activeKEK, err := crypto.LoadKEKsFromEnv()
	if err != nil {
		db.Close()
		return Runtime{}, err
	}
	publicBaseURL := cfg.PublicBaseURL
	if publicBaseURL == "" {
		publicBaseURL = cfg.BaseURL
	}
	svc := services.NewWithKEKs(services.Dependencies{
		Projects:           stores,
		Notes:              noteStore{stores},
		Artifacts:          artifactStore{stores},
		Decisions:          decisionStore{stores},
		OpenQuestions:      openQuestionStore{stores},
		Packets:            packetStore{stores},
		APIKeys:            apiKeyStore{stores},
		JudgmentTraces:     stores,
		HeuristicProposals: stores,
		ApprovedHeuristics: stores,
		PacketSnapshots:    stores,
		Idempotency:        stores,
		CuratorJobs:        stores,
		OGImages:           ogWriter,
		CacheInvalidator:   services.NoopCacheInvalidator{},
		PublicBaseURL:      publicBaseURL,
		Users:              stores,
		OAuthIdentities:    stores,
		UserSessions:       stores,
		OAuthStates:        stores,
		Onboarding:         stores,
		ProviderKeys:       stores,
	}, keks, activeKEK)

	return Runtime{Services: svc, APIKeys: apiKeyStore{stores}}, nil
}

type noteStore struct{ postgres.Stores }

func (s noteStore) CreateNote(ctx context.Context, note domain.Note) (domain.Note, error) {
	return s.Stores.CreateNote(ctx, note)
}

func (s noteStore) CountByProject(ctx context.Context, projectID string) (int, error) {
	return s.Stores.CountNotesByProject(ctx, projectID)
}

func (s noteStore) ListByProject(ctx context.Context, projectID string) ([]domain.Note, error) {
	return s.Stores.ListNotesByProject(ctx, projectID)
}

type artifactStore struct{ postgres.Stores }

func (s artifactStore) CreateArtifact(ctx context.Context, artifact domain.Artifact) (domain.Artifact, error) {
	return s.Stores.CreateArtifact(ctx, artifact)
}

func (s artifactStore) CountByProject(ctx context.Context, projectID string) (int, error) {
	return s.Stores.CountArtifactsByProject(ctx, projectID)
}

func (s artifactStore) ListByProject(ctx context.Context, projectID string) ([]domain.Artifact, error) {
	return s.Stores.ListArtifactsByProject(ctx, projectID)
}

type decisionStore struct{ postgres.Stores }

func (s decisionStore) CreateDecision(ctx context.Context, decision domain.Decision) (domain.Decision, error) {
	return s.Stores.CreateDecision(ctx, decision)
}

func (s decisionStore) CountByProject(ctx context.Context, projectID string) (int, error) {
	return s.Stores.CountDecisionsByProject(ctx, projectID)
}

func (s decisionStore) ListByProject(ctx context.Context, projectID string) ([]domain.Decision, error) {
	return s.Stores.ListDecisionsByProject(ctx, projectID)
}

type openQuestionStore struct{ postgres.Stores }

func (s openQuestionStore) CreateOpenQuestion(ctx context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error) {
	return s.Stores.CreateOpenQuestion(ctx, question)
}

func (s openQuestionStore) CountByProject(ctx context.Context, projectID string) (int, error) {
	return s.Stores.CountOpenQuestionsByProject(ctx, projectID)
}

func (s openQuestionStore) ListByProject(ctx context.Context, projectID string) ([]domain.OpenQuestion, error) {
	return s.Stores.ListOpenQuestionsByProject(ctx, projectID)
}

type packetStore struct{ postgres.Stores }

func (s packetStore) CreatePacket(ctx context.Context, packet domain.Packet) (domain.Packet, error) {
	return s.Stores.CreatePacket(ctx, packet)
}

func (s packetStore) LatestByProject(ctx context.Context, projectID string) (domain.Packet, error) {
	return s.Stores.LatestByProject(ctx, projectID)
}

type apiKeyStore struct{ postgres.Stores }

func (s apiKeyStore) CreateAPIKey(ctx context.Context, key domain.APIKey) (domain.APIKey, error) {
	return s.Stores.CreateAPIKey(ctx, key)
}

func (s apiKeyStore) GetByTokenHash(ctx context.Context, tokenHash string) (domain.APIKey, error) {
	return s.Stores.GetByTokenHash(ctx, tokenHash)
}

func (s apiKeyStore) ListAPIKeys(ctx context.Context) ([]domain.APIKey, error) {
	return s.Stores.ListAPIKeys(ctx)
}

func (s apiKeyStore) ListAPIKeysByOwner(ctx context.Context, userID string) ([]domain.APIKey, error) {
	return s.Stores.ListAPIKeysByOwner(ctx, userID)
}

func (s apiKeyStore) RevokeAPIKey(ctx context.Context, keyID string) (domain.APIKey, error) {
	return s.Stores.RevokeAPIKey(ctx, keyID)
}

func (s apiKeyStore) RevokeAPIKeyByOwner(ctx context.Context, userID string, keyID string) (domain.APIKey, error) {
	return s.Stores.RevokeAPIKeyByOwner(ctx, userID, keyID)
}
