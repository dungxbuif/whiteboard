CREATE TABLE workspace_domains (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    domain text NOT NULL,
    status text NOT NULL CHECK (status IN ('pending','verified','captured','revoked')),
    challenge_hash bytea NOT NULL CHECK (octet_length(challenge_hash) = 32),
    challenge_expires_at timestamptz NOT NULL,
    verified_at timestamptz,
    captured_at timestamptz,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT workspace_domains_normalized_check CHECK (domain = lower(domain) AND domain !~ '\\.$'),
    CONSTRAINT workspace_domains_domain_exact_key UNIQUE (domain),
    CONSTRAINT workspace_domains_workspace_id_id_key UNIQUE (workspace_id,id),
    CONSTRAINT workspace_domains_verified_state_check CHECK ((status = 'pending' AND verified_at IS NULL) OR status <> 'pending'),
    CONSTRAINT workspace_domains_captured_state_check CHECK ((status = 'captured' AND captured_at IS NOT NULL) OR status <> 'captured')
);

CREATE INDEX workspace_domains_workspace_created_idx ON workspace_domains(workspace_id,created_at,id);

CREATE TABLE workspace_domain_requests (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    request_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('begin','verify','capture','revoke')),
    principal_kind text NOT NULL,
    principal_id text NOT NULL,
    payload_hash char(64) NOT NULL CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    response_status integer NOT NULL CHECK (response_status BETWEEN 100 AND 599),
    response_nonce bytea NOT NULL CHECK (octet_length(response_nonce) = 12),
    response_ciphertext bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (workspace_id,request_id)
);
