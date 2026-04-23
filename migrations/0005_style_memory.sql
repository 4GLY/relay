DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM projects
    GROUP BY name
    HAVING COUNT(*) > 1
  ) THEN
    RAISE EXCEPTION 'cannot add unique project name index while duplicate project names exist';
  END IF;
END
$$;

CREATE UNIQUE INDEX IF NOT EXISTS projects_name_unique_idx ON projects (name);
CREATE INDEX IF NOT EXISTS projects_root_path_idx ON projects (root_path) WHERE root_path IS NOT NULL;

CREATE TABLE IF NOT EXISTS judgment_traces (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  workflow TEXT NOT NULL,
  artifact_type TEXT NOT NULL,
  decision TEXT NOT NULL,
  alternatives JSONB NOT NULL DEFAULT '[]'::jsonb,
  rationale TEXT NOT NULL,
  constraints JSONB NOT NULL DEFAULT '[]'::jsonb,
  source_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
  language TEXT NOT NULL DEFAULT 'unknown',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS judgment_traces_project_created_idx
  ON judgment_traces (project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS judgment_traces_project_workflow_artifact_idx
  ON judgment_traces (project_id, workflow, artifact_type, created_at DESC);

CREATE TABLE IF NOT EXISTS heuristic_proposals (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  origin_trace_id TEXT REFERENCES judgment_traces(id) ON DELETE SET NULL,
  workflow TEXT NOT NULL DEFAULT '',
  artifact_type TEXT NOT NULL DEFAULT '',
  heuristic_key TEXT NOT NULL,
  canonical_text TEXT NOT NULL,
  normalized_text TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'pending',
  source_trace_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  source_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
  proposed_by TEXT NOT NULL DEFAULT '',
  review_notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS heuristic_proposals_project_state_created_idx
  ON heuristic_proposals (project_id, state, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS heuristic_proposals_project_key_state_idx
  ON heuristic_proposals (project_id, heuristic_key, state)
  WHERE state = 'pending';

CREATE TABLE IF NOT EXISTS approved_heuristics (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  origin_proposal_id TEXT REFERENCES heuristic_proposals(id) ON DELETE SET NULL,
  workflow TEXT NOT NULL DEFAULT '',
  artifact_type TEXT NOT NULL DEFAULT '',
  heuristic_key TEXT NOT NULL,
  canonical_text TEXT NOT NULL,
  state TEXT NOT NULL DEFAULT 'approved',
  source_trace_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  source_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS approved_heuristics_project_state_created_idx
  ON approved_heuristics (project_id, state, created_at DESC);
CREATE INDEX IF NOT EXISTS approved_heuristics_scope_idx
  ON approved_heuristics (project_id, state, workflow, artifact_type, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS approved_heuristics_project_key_active_idx
  ON approved_heuristics (project_id, heuristic_key)
  WHERE state = 'approved';

CREATE TABLE IF NOT EXISTS packet_snapshots (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  packet_kind TEXT NOT NULL,
  target TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  task_summary TEXT NOT NULL DEFAULT '',
  rendered_body TEXT NOT NULL,
  approved_heuristic_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  decision_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  open_question_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  source_artifact_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  missing_context JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS packet_snapshots_project_created_idx
  ON packet_snapshots (project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS idempotency_records (
  id TEXT PRIMARY KEY,
  scope_kind TEXT NOT NULL,
  scope_project_id TEXT NOT NULL DEFAULT '',
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  response_kind TEXT NOT NULL,
  response_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (scope_kind, scope_project_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS curator_jobs (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  job_kind TEXT NOT NULL,
  state TEXT NOT NULL DEFAULT 'pending',
  dedupe_key TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  lease_owner TEXT NOT NULL DEFAULT '',
  lease_expires_at TIMESTAMPTZ,
  available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS curator_jobs_dedupe_active_idx
  ON curator_jobs (dedupe_key)
  WHERE state IN ('pending', 'leased');
CREATE INDEX IF NOT EXISTS curator_jobs_claim_idx
  ON curator_jobs (state, available_at, lease_expires_at, created_at);
