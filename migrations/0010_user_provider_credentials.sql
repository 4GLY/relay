-- migrations/0010_user_provider_credentials.sql
-- Provider credentials are settings-owned state, not onboarding state.

CREATE TABLE IF NOT EXISTS user_provider_credentials (
  user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider       TEXT NOT NULL,
  key_ciphertext BYTEA NOT NULL,
  key_nonce      BYTEA NOT NULL,
  key_kek_version SMALLINT NOT NULL DEFAULT 1,
  key_prefix     TEXT NOT NULL DEFAULT '',
  key_last4      TEXT NOT NULL DEFAULT '',
  aad_salt       BYTEA NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at     TIMESTAMPTZ,
  PRIMARY KEY (user_id, provider)
);

CREATE INDEX IF NOT EXISTS user_provider_credentials_user_active_idx
  ON user_provider_credentials (user_id)
  WHERE deleted_at IS NULL;

-- One-time compatibility lift for any key material captured before provider
-- credentials were split from onboarding. The legacy columns stay in place for
-- rollback safety, but new writes go only to user_provider_credentials.
INSERT INTO user_provider_credentials (
  user_id,
  provider,
  key_ciphertext,
  key_nonce,
  key_kek_version,
  key_prefix,
  key_last4,
  aad_salt,
  created_at,
  updated_at
)
SELECT
  user_id,
  'anthropic',
  anthropic_key_ciphertext,
  anthropic_key_nonce,
  anthropic_key_kek_version,
  anthropic_key_prefix,
  anthropic_key_last4,
  aad_salt,
  created_at,
  updated_at
FROM user_onboarding
WHERE anthropic_key_ciphertext IS NOT NULL
  AND anthropic_key_nonce IS NOT NULL
  AND aad_salt IS NOT NULL
ON CONFLICT (user_id, provider) DO NOTHING;
