package services

import (
	"context"

	"relay/internal/domain"
	"relay/internal/lib"
)

func summarizeAPIKey(key domain.APIKey) APIKeySummary {
	return APIKeySummary{
		KeyID:       key.ID,
		Name:        key.Name,
		TokenPrefix: key.TokenPrefix,
		Scope:       NormalizeAPIKeyScope(key.Scope),
		ProjectID:   key.ProjectID,
		Revoked:     key.Revoked,
	}
}

func summarizeRevokedAPIKey(key domain.APIKey) RevokeAPIKeyResult {
	return RevokeAPIKeyResult{
		KeyID:       key.ID,
		Name:        key.Name,
		TokenPrefix: key.TokenPrefix,
		Scope:       NormalizeAPIKeyScope(key.Scope),
		ProjectID:   key.ProjectID,
		Revoked:     key.Revoked,
	}
}

func (s Service) IssueAPIKey(ctx context.Context, input IssueAPIKeyInput) (IssueAPIKeyResult, error) {
	if err := requireAdminAuth(ctx); err != nil {
		return IssueAPIKeyResult{}, err
	}
	if err := validateIssueAPIKeyInput(input); err != nil {
		return IssueAPIKeyResult{}, err
	}
	if input.Name == "" {
		return IssueAPIKeyResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "name")
	}
	if s.deps.APIKeys == nil {
		return IssueAPIKeyResult{}, lib.Misconfigured("api key store is required")
	}

	scope := NormalizeAPIKeyScope(input.Scope)
	if scope != APIKeyScopeGlobal && scope != APIKeyScopeProject {
		return IssueAPIKeyResult{}, lib.AppError{
			Code:      "INVALID_API_KEY_SCOPE",
			Message:   "api key scope must be global or project",
			Retryable: false,
		}
	}

	boundProjectID := ""
	if scope == APIKeyScopeProject {
		project, err := s.resolveProject(ctx, input.Project, input.ProjectID)
		if err != nil {
			return IssueAPIKeyResult{}, err
		}
		boundProjectID = project.ID
	}

	token, err := lib.NewSecretToken("relay_live")
	if err != nil {
		return IssueAPIKeyResult{}, err
	}

	key := domain.APIKey{
		ID:          lib.NewID("key"),
		Name:        input.Name,
		TokenHash:   lib.TokenHash(token),
		TokenPrefix: lib.TokenPrefix(token),
		Scope:       scope,
		ProjectID:   boundProjectID,
	}

	created, err := s.deps.APIKeys.CreateAPIKey(ctx, key)
	if err != nil {
		return IssueAPIKeyResult{}, err
	}

	return IssueAPIKeyResult{
		KeyID:       created.ID,
		Name:        created.Name,
		Token:       token,
		TokenPrefix: created.TokenPrefix,
		Scope:       NormalizeAPIKeyScope(created.Scope),
		ProjectID:   created.ProjectID,
	}, nil
}

func (s Service) ListAPIKeys(ctx context.Context) (ListAPIKeysResult, error) {
	if err := requireAdminAuth(ctx); err != nil {
		return ListAPIKeysResult{}, err
	}
	if s.deps.APIKeys == nil {
		return ListAPIKeysResult{}, lib.Misconfigured("api key store is required")
	}

	keys, err := s.deps.APIKeys.ListAPIKeys(ctx)
	if err != nil {
		return ListAPIKeysResult{}, err
	}

	items := make([]APIKeySummary, 0, len(keys))
	for _, key := range keys {
		items = append(items, summarizeAPIKey(key))
	}

	return ListAPIKeysResult{Items: items}, nil
}

func (s Service) RevokeAPIKey(ctx context.Context, input RevokeAPIKeyInput) (RevokeAPIKeyResult, error) {
	if err := requireAdminAuth(ctx); err != nil {
		return RevokeAPIKeyResult{}, err
	}
	if err := validateRevokeAPIKeyInput(input); err != nil {
		return RevokeAPIKeyResult{}, err
	}
	if input.KeyID == "" {
		return RevokeAPIKeyResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "key_id")
	}
	if s.deps.APIKeys == nil {
		return RevokeAPIKeyResult{}, lib.Misconfigured("api key store is required")
	}

	key, err := s.deps.APIKeys.RevokeAPIKey(ctx, input.KeyID)
	if err != nil {
		return RevokeAPIKeyResult{}, err
	}

	return summarizeRevokedAPIKey(key), nil
}

func (s Service) IssueUserAPIKey(ctx context.Context, input IssueAPIKeyInput) (IssueAPIKeyResult, error) {
	auth, err := RequireUserAuth(ctx)
	if err != nil {
		return IssueAPIKeyResult{}, err
	}
	if err := validateStringFieldLength("name", input.Name, maxAPIKeyNameLength); err != nil {
		return IssueAPIKeyResult{}, err
	}
	if input.Name == "" {
		return IssueAPIKeyResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "name")
	}
	if input.Scope != "" || input.Project != "" || input.ProjectID != "" {
		return IssueAPIKeyResult{}, lib.AppError{
			Code:      "INVALID_USER_API_KEY_INPUT",
			Message:   "user API keys accept only name",
			Retryable: false,
		}
	}
	if s.deps.APIKeys == nil {
		return IssueAPIKeyResult{}, lib.Misconfigured("api key store is required")
	}

	token, err := lib.NewSecretToken("relay_live")
	if err != nil {
		return IssueAPIKeyResult{}, err
	}

	key := domain.APIKey{
		ID:          lib.NewID("key"),
		Name:        input.Name,
		TokenHash:   lib.TokenHash(token),
		TokenPrefix: lib.TokenPrefix(token),
		Scope:       APIKeyScopeGlobal,
		OwnerUserID: auth.UserID,
	}

	created, err := s.deps.APIKeys.CreateAPIKey(ctx, key)
	if err != nil {
		return IssueAPIKeyResult{}, err
	}

	return IssueAPIKeyResult{
		KeyID:       created.ID,
		Name:        created.Name,
		Token:       token,
		TokenPrefix: created.TokenPrefix,
		Scope:       NormalizeAPIKeyScope(created.Scope),
		ProjectID:   created.ProjectID,
	}, nil
}

func (s Service) ListUserAPIKeys(ctx context.Context) (ListAPIKeysResult, error) {
	auth, err := RequireUserAuth(ctx)
	if err != nil {
		return ListAPIKeysResult{}, err
	}
	if s.deps.APIKeys == nil {
		return ListAPIKeysResult{}, lib.Misconfigured("api key store is required")
	}

	keys, err := s.deps.APIKeys.ListAPIKeysByOwner(ctx, auth.UserID)
	if err != nil {
		return ListAPIKeysResult{}, err
	}

	items := make([]APIKeySummary, 0, len(keys))
	for _, key := range keys {
		items = append(items, summarizeAPIKey(key))
	}

	return ListAPIKeysResult{Items: items}, nil
}

func (s Service) RevokeUserAPIKey(ctx context.Context, input RevokeAPIKeyInput) (RevokeAPIKeyResult, error) {
	auth, err := RequireUserAuth(ctx)
	if err != nil {
		return RevokeAPIKeyResult{}, err
	}
	if err := validateRevokeAPIKeyInput(input); err != nil {
		return RevokeAPIKeyResult{}, err
	}
	if input.KeyID == "" {
		return RevokeAPIKeyResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "key_id")
	}
	if s.deps.APIKeys == nil {
		return RevokeAPIKeyResult{}, lib.Misconfigured("api key store is required")
	}

	key, err := s.deps.APIKeys.RevokeAPIKeyByOwner(ctx, auth.UserID, input.KeyID)
	if err != nil {
		return RevokeAPIKeyResult{}, err
	}

	return summarizeRevokedAPIKey(key), nil
}
