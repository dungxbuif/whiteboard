CREATE TABLE feature_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    event_id BIGINT,
    kind TEXT NOT NULL CHECK (length(kind) BETWEEN 1 AND 128),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload) = 'object'),
    idempotency_key TEXT,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_owner TEXT,
    lease_expires_at TIMESTAMPTZ,
    last_error TEXT,
    completed_at TIMESTAMPTZ,
    dead_lettered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT feature_jobs_lease_pair CHECK ((lease_owner IS NULL) = (lease_expires_at IS NULL)),
    CONSTRAINT feature_jobs_terminal_state CHECK (completed_at IS NULL OR dead_lettered_at IS NULL),
    CONSTRAINT feature_jobs_idempotency UNIQUE (workspace_id, kind, idempotency_key)
);

CREATE INDEX feature_jobs_claim_idx
    ON feature_jobs (available_at, id)
    WHERE completed_at IS NULL AND dead_lettered_at IS NULL;
