package services

import (
	"context"
	"testing"
	"time"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/lib/crypto"
)

type fakeProviderCredentialStore struct {
	rows map[string]domain.UserProviderCredential
}

func newFakeProviderCredentialStore() *fakeProviderCredentialStore {
	return &fakeProviderCredentialStore{rows: map[string]domain.UserProviderCredential{}}
}

func providerCredentialKey(userID string, provider string) string {
	return userID + ":" + provider
}

func (s *fakeProviderCredentialStore) UpsertUserProviderCredential(_ context.Context, credential domain.UserProviderCredential) (domain.UserProviderCredential, error) {
	now := time.Now()
	key := providerCredentialKey(credential.UserID, credential.Provider)
	if existing, ok := s.rows[key]; ok {
		credential.CreatedAt = existing.CreatedAt
	} else {
		credential.CreatedAt = now
	}
	credential.UpdatedAt = now
	credential.DeletedAt = nil
	s.rows[key] = credential
	return credential, nil
}

func (s *fakeProviderCredentialStore) GetUserProviderCredential(_ context.Context, userID string, provider string) (domain.UserProviderCredential, error) {
	row, ok := s.rows[providerCredentialKey(userID, provider)]
	if !ok || row.DeletedAt != nil {
		return domain.UserProviderCredential{}, lib.NotFound("PROVIDER_CREDENTIAL_NOT_FOUND", "provider credential not found")
	}
	return row, nil
}

func (s *fakeProviderCredentialStore) ListUserProviderCredentials(_ context.Context, userID string) ([]domain.UserProviderCredential, error) {
	var rows []domain.UserProviderCredential
	for _, row := range s.rows {
		if row.UserID == userID && row.DeletedAt == nil {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (s *fakeProviderCredentialStore) DeleteUserProviderCredential(_ context.Context, userID string, provider string) error {
	key := providerCredentialKey(userID, provider)
	row, ok := s.rows[key]
	if !ok || row.DeletedAt != nil {
		return lib.NotFound("PROVIDER_CREDENTIAL_NOT_FOUND", "provider credential not found")
	}
	now := time.Now()
	row.DeletedAt = &now
	row.UpdatedAt = now
	s.rows[key] = row
	return nil
}

func newProviderCredentialService(store *fakeProviderCredentialStore) Service {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return NewWithKEKs(
		Dependencies{ProviderKeys: store},
		map[crypto.KEKVersion][]byte{1: key},
		1,
	)
}

func TestUpsertProviderCredentialStoresEncryptedKeyOutsideOnboarding(t *testing.T) {
	store := newFakeProviderCredentialStore()
	svc := newProviderCredentialService(store)

	result, err := svc.UpsertProviderCredential(authedCtxFor("usr_provider"), ProviderCredentialUpsertInput{
		Provider: "anthropic",
		APIKey:   "sk-ant-test-provider-1234",
	})
	if err != nil {
		t.Fatalf("UpsertProviderCredential: %v", err)
	}

	if !result.Connected || result.Provider != "anthropic" || result.KeyLast4 != "1234" {
		t.Fatalf("unexpected result: %#v", result)
	}
	row := store.rows[providerCredentialKey("usr_provider", "anthropic")]
	if len(row.KeyCiphertext) == 0 || len(row.KeyNonce) == 0 || len(row.AadSalt) != 16 {
		t.Fatalf("expected encrypted provider credential material, got %#v", row)
	}
	if string(row.KeyCiphertext) == "sk-ant-test-provider-1234" {
		t.Fatal("ciphertext contains raw provider key")
	}
}

func TestProviderCredentialRequiresKEK(t *testing.T) {
	svc := New(Dependencies{ProviderKeys: newFakeProviderCredentialStore()})
	_, err := svc.UpsertProviderCredential(authedCtxFor("usr_provider"), ProviderCredentialUpsertInput{
		Provider: "anthropic",
		APIKey:   "sk-ant-test-provider-1234",
	})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "MISCONFIGURED" {
		t.Fatalf("expected MISCONFIGURED, got %#v", err)
	}
}

func TestProviderCredentialRejectsUnsupportedProvider(t *testing.T) {
	svc := newProviderCredentialService(newFakeProviderCredentialStore())
	_, err := svc.UpsertProviderCredential(authedCtxFor("usr_provider"), ProviderCredentialUpsertInput{
		Provider: "openai",
		APIKey:   "sk-test",
	})
	appErr, ok := err.(lib.AppError)
	if !ok || appErr.Code != "UNSUPPORTED_PROVIDER" {
		t.Fatalf("expected UNSUPPORTED_PROVIDER, got %#v", err)
	}
}

func TestDeleteProviderCredentialRemovesFromStatusList(t *testing.T) {
	store := newFakeProviderCredentialStore()
	svc := newProviderCredentialService(store)
	ctx := authedCtxFor("usr_provider")

	if _, err := svc.UpsertProviderCredential(ctx, ProviderCredentialUpsertInput{Provider: "anthropic", APIKey: "sk-ant-test-provider-1234"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	before, err := svc.ListProviderCredentials(ctx)
	if err != nil {
		t.Fatalf("list before delete: %v", err)
	}
	if len(before.Credentials) != 1 {
		t.Fatalf("expected one credential before delete, got %#v", before)
	}
	if err := svc.DeleteProviderCredential(ctx, "anthropic"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	after, err := svc.ListProviderCredentials(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(after.Credentials) != 0 {
		t.Fatalf("expected no credentials after delete, got %#v", after)
	}
}
