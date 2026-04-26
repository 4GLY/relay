package services

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/crypto"
)

const providerAnthropic = "anthropic"

type ProviderCredentialUpsertInput struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
}

type ProviderCredentialStatus struct {
	Provider  string    `json:"provider"`
	Connected bool      `json:"connected"`
	KeyPrefix string    `json:"key_prefix,omitempty"`
	KeyLast4  string    `json:"key_last4,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type ProviderCredentialListResult struct {
	Credentials []ProviderCredentialStatus `json:"credentials"`
}

func (s Service) UpsertProviderCredential(ctx context.Context, input ProviderCredentialUpsertInput) (ProviderCredentialStatus, error) {
	auth, err := RequireUserAuth(ctx)
	if err != nil {
		return ProviderCredentialStatus{}, err
	}
	if auth.UserID == "" {
		return ProviderCredentialStatus{}, lib.Forbidden("UNAUTHORIZED", "provider credentials require a user session")
	}
	if s.deps.ProviderKeys == nil {
		return ProviderCredentialStatus{}, lib.Misconfigured("provider credential store is required")
	}
	if len(s.keks) == 0 || s.activeKEKVersion == 0 {
		return ProviderCredentialStatus{}, lib.Misconfigured("data encryption key is not configured")
	}

	provider, err := normalizeProvider(input.Provider)
	if err != nil {
		return ProviderCredentialStatus{}, err
	}
	rawKey := strings.TrimSpace(input.APIKey)
	if rawKey == "" {
		return ProviderCredentialStatus{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "api_key")
	}
	if provider == providerAnthropic && !strings.HasPrefix(rawKey, "sk-ant-") {
		return ProviderCredentialStatus{}, lib.Forbidden("INVALID_PROVIDER_CREDENTIAL", "Anthropic keys must start with sk-ant-")
	}

	aadSalt := make([]byte, 16)
	if _, err := rand.Read(aadSalt); err != nil {
		return ProviderCredentialStatus{}, err
	}
	env, err := crypto.Encrypt(s.keks, s.activeKEKVersion, []byte(rawKey), aadSalt)
	if err != nil {
		return ProviderCredentialStatus{}, err
	}

	saved, err := s.deps.ProviderKeys.UpsertUserProviderCredential(ctx, domain.UserProviderCredential{
		UserID:        auth.UserID,
		Provider:      provider,
		KeyCiphertext: env.Ciphertext,
		KeyNonce:      env.Nonce,
		KeyKEKVersion: uint8(env.KEKVersion),
		KeyPrefix:     keyPrefix(rawKey),
		KeyLast4:      keyLast4(rawKey),
		AadSalt:       aadSalt,
	})
	if err != nil {
		return ProviderCredentialStatus{}, err
	}
	return providerCredentialStatus(saved), nil
}

func (s Service) ListProviderCredentials(ctx context.Context) (ProviderCredentialListResult, error) {
	auth, err := RequireUserAuth(ctx)
	if err != nil {
		return ProviderCredentialListResult{}, err
	}
	if auth.UserID == "" {
		return ProviderCredentialListResult{}, lib.Forbidden("UNAUTHORIZED", "provider credentials require a user session")
	}
	if s.deps.ProviderKeys == nil {
		return ProviderCredentialListResult{}, lib.Misconfigured("provider credential store is required")
	}
	rows, err := s.deps.ProviderKeys.ListUserProviderCredentials(ctx, auth.UserID)
	if err != nil {
		return ProviderCredentialListResult{}, err
	}
	items := make([]ProviderCredentialStatus, 0, len(rows))
	for _, row := range rows {
		items = append(items, providerCredentialStatus(row))
	}
	return ProviderCredentialListResult{Credentials: items}, nil
}

func (s Service) DeleteProviderCredential(ctx context.Context, providerRaw string) error {
	auth, err := RequireUserAuth(ctx)
	if err != nil {
		return err
	}
	if auth.UserID == "" {
		return lib.Forbidden("UNAUTHORIZED", "provider credentials require a user session")
	}
	if s.deps.ProviderKeys == nil {
		return lib.Misconfigured("provider credential store is required")
	}
	provider, err := normalizeProvider(providerRaw)
	if err != nil {
		return err
	}
	return s.deps.ProviderKeys.DeleteUserProviderCredential(ctx, auth.UserID, provider)
}

func providerCredentialStatus(row domain.UserProviderCredential) ProviderCredentialStatus {
	return ProviderCredentialStatus{
		Provider:  row.Provider,
		Connected: row.DeletedAt == nil,
		KeyPrefix: row.KeyPrefix,
		KeyLast4:  row.KeyLast4,
		UpdatedAt: row.UpdatedAt,
	}
}

func normalizeProvider(raw string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(raw))
	if provider == "" {
		return "", lib.MissingFields("MISSING_REQUIRED_FIELDS", "provider")
	}
	if provider != providerAnthropic {
		return "", lib.Forbidden("UNSUPPORTED_PROVIDER", "provider is not supported")
	}
	return provider, nil
}
