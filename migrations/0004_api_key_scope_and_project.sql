ALTER TABLE api_keys
  ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'global',
  ADD COLUMN IF NOT EXISTS project_id TEXT REFERENCES projects(id) ON DELETE CASCADE;

UPDATE api_keys
SET scope = 'global'
WHERE scope IS NULL OR scope = '';

ALTER TABLE api_keys
  DROP CONSTRAINT IF EXISTS api_keys_scope_check;

ALTER TABLE api_keys
  ADD CONSTRAINT api_keys_scope_check
  CHECK (
    scope IN ('global', 'project')
    AND (scope <> 'project' OR project_id IS NOT NULL)
  );
