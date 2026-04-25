-- V2 foundation: end-user accounts + multi-provider OAuth (GitHub, Google).
-- CEO plan refs: A1 (User model), Open Q #1, #2. Eng review default seeds locked.

CREATE TABLE IF NOT EXISTS users (
  id            TEXT PRIMARY KEY,
  email         TEXT,
  display_name  TEXT NOT NULL DEFAULT '',
  avatar_url    TEXT NOT NULL DEFAULT '',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
  ON users (LOWER(email))
  WHERE email IS NOT NULL;

CREATE TABLE IF NOT EXISTS oauth_identities (
  id                TEXT PRIMARY KEY,
  user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider          TEXT NOT NULL,
  provider_user_id  TEXT NOT NULL,
  provider_login    TEXT NOT NULL DEFAULT '',
  verified_email    TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS oauth_identities_provider_user_idx
  ON oauth_identities (provider, provider_user_id);

CREATE INDEX IF NOT EXISTS oauth_identities_user_idx
  ON oauth_identities (user_id);

CREATE TABLE IF NOT EXISTS user_sessions (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS user_sessions_token_hash_idx
  ON user_sessions (token_hash);

CREATE INDEX IF NOT EXISTS user_sessions_user_expires_idx
  ON user_sessions (user_id, expires_at);

CREATE TABLE IF NOT EXISTS oauth_states (
  id          TEXT PRIMARY KEY,
  provider    TEXT NOT NULL,
  redirect_to TEXT NOT NULL DEFAULT '',
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  consumed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS oauth_states_expires_idx
  ON oauth_states (expires_at);

ALTER TABLE projects
  ADD COLUMN IF NOT EXISTS owner_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS projects_owner_user_idx
  ON projects (owner_user_id) WHERE owner_user_id IS NOT NULL;

ALTER TABLE api_keys
  ADD COLUMN IF NOT EXISTS owner_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS api_keys_owner_user_idx
  ON api_keys (owner_user_id) WHERE owner_user_id IS NOT NULL;
