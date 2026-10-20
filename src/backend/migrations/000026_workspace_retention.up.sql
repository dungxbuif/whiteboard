CREATE TABLE workspace_retention_policies (
  workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
  revision bigint NOT NULL DEFAULT 0 CHECK (revision >= 0),
  policy jsonb NOT NULL DEFAULT '{"activityDays":90,"notificationDays":30,"automaticVersionLimit":100,"trashHardPurge":false}'::jsonb CHECK (jsonb_typeof(policy)='object'),
  policy_hash text NOT NULL DEFAULT repeat('0',64) CHECK (policy_hash ~ '^[a-f0-9]{64}$'),
  effective_at timestamptz NOT NULL DEFAULT 'infinity',
  scheduled_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE retention_previews (
  id uuid PRIMARY KEY,
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  policy_revision bigint NOT NULL CHECK (policy_revision >= 0),
  policy jsonb NOT NULL CHECK (jsonb_typeof(policy)='object'),
  policy_hash text NOT NULL CHECK (policy_hash ~ '^[a-f0-9]{64}$'),
  counts jsonb NOT NULL CHECK (jsonb_typeof(counts)='object'),
  earliest_destructive_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  UNIQUE (workspace_id,id)
);
CREATE INDEX retention_previews_expiry ON retention_previews(expires_at);

CREATE TABLE retention_holds (
  id uuid PRIMARY KEY,
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('all','activity','notification','version','comment')),
  ref_id text NOT NULL CHECK (length(ref_id) BETWEEN 1 AND 256),
  version_id uuid,
  version_key uuid GENERATED ALWAYS AS (coalesce(version_id,'00000000-0000-0000-0000-000000000000'::uuid)) STORED,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  UNIQUE (workspace_id,kind,ref_id,version_key)
);
CREATE INDEX retention_holds_lookup ON retention_holds(workspace_id,kind,ref_id);

CREATE TABLE retention_mutation_receipts (
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  request_id uuid NOT NULL,
  principal_kind text NOT NULL,
  principal_id text NOT NULL,
  action text NOT NULL CHECK (action IN ('retention.schedule','retention.hold')),
  payload_hash text NOT NULL CHECK (payload_hash ~ '^[a-f0-9]{64}$'),
  response_status integer NOT NULL CHECK (response_status BETWEEN 200 AND 299),
  response_body bytea NOT NULL,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (workspace_id,request_id)
);

CREATE TABLE retention_purge_audit (
  id bigserial PRIMARY KEY,
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  policy_revision bigint NOT NULL,
  kind text NOT NULL CHECK (kind IN ('activity','notification','version','comment')),
  subject_id text NOT NULL CHECK (length(subject_id) BETWEEN 1 AND 256),
  purged_at timestamptz NOT NULL,
  UNIQUE (workspace_id,kind,subject_id)
);
CREATE INDEX retention_purge_audit_workspace ON retention_purge_audit(workspace_id,purged_at DESC,id DESC);

