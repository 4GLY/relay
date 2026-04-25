ALTER TABLE packet_snapshots
  ADD COLUMN IF NOT EXISTS public_readable BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS public_token TEXT,
  ADD COLUMN IF NOT EXISTS og_image_path TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS packet_snapshots_public_token_idx
  ON packet_snapshots (public_token)
  WHERE public_readable = TRUE AND public_token IS NOT NULL;
