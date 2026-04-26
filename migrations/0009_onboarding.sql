-- migrations/0009_onboarding.sql
-- V2 S4: per-user onboarding state plus optional encrypted Anthropic key material.

-- E1: unique index for EnsureProjectByOwnerName idempotency (D4).
-- Full index (no WHERE): partial indexes require the matching predicate in ON
-- CONFLICT, which pgx does not support cleanly. owner_user_id is always set for
-- rows we insert (FK to users), so a full index is equivalent here.
CREATE UNIQUE INDEX IF NOT EXISTS projects_owner_name_uniq
  ON projects (owner_user_id, name);

CREATE TABLE IF NOT EXISTS user_onboarding (
  user_id                   TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  -- Optional provider key material: NULL for keyless onboarding and after key deletion.
  anthropic_key_ciphertext  BYTEA,
  anthropic_key_nonce       BYTEA,
  anthropic_key_kek_version SMALLINT NOT NULL DEFAULT 1,
  anthropic_key_prefix      TEXT NOT NULL DEFAULT '',
  anthropic_key_last4       TEXT NOT NULL DEFAULT '',
  -- D6: per-row random AAD salt; generated in Go (E2: no pgcrypto). NULL after key deletion.
  aad_salt                  BYTEA,
  default_project_id        TEXT REFERENCES projects(id) ON DELETE SET NULL,
  onboarding_completed_at   TIMESTAMPTZ,            -- NULL = first-run onboarding not completed
  last_validated_at         TIMESTAMPTZ,
  created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS user_onboarding_completed_idx
  ON user_onboarding (onboarding_completed_at)
  WHERE onboarding_completed_at IS NOT NULL;
