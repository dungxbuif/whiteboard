CREATE TABLE workspace_sso_connections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL UNIQUE REFERENCES workspaces(id) ON DELETE CASCADE,
    issuer text NOT NULL,
    client_id text NOT NULL,
    client_secret_cipher bytea NOT NULL,
    mode text NOT NULL CHECK (mode IN ('optional','enforced')),
    public_links_enabled boolean NOT NULL,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT workspace_sso_issuer_https CHECK (issuer LIKE 'https://%' AND issuer !~ '/$')
);

CREATE UNIQUE INDEX workspace_sso_connections_issuer_client_key
    ON workspace_sso_connections (issuer,client_id);

CREATE TABLE sso_flow_transactions (
    state_hash bytea PRIMARY KEY CHECK (octet_length(state_hash)=32),
    connection_id uuid NOT NULL REFERENCES workspace_sso_connections(id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    browser_binding_hash bytea NOT NULL CHECK (octet_length(browser_binding_hash)=32),
    nonce_hash bytea NOT NULL CHECK (octet_length(nonce_hash)=32),
    pkce_verifier_cipher bytea NOT NULL,
    return_path text NOT NULL,
    link_user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    link_request_id uuid,
    source_session_id uuid,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz
);

ALTER TABLE sso_flow_transactions ADD CONSTRAINT sso_flow_link_pair
    CHECK ((link_user_id IS NULL) = (link_request_id IS NULL) AND (link_user_id IS NULL) = (source_session_id IS NULL));

CREATE INDEX sso_flow_transactions_expiry_idx ON sso_flow_transactions(expires_at);

CREATE TABLE sso_external_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id uuid NOT NULL REFERENCES workspace_sso_connections(id) ON DELETE CASCADE,
    issuer text NOT NULL,
    subject text NOT NULL,
    email text NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (issuer,subject),
    UNIQUE (connection_id,user_id)
);

CREATE TABLE enterprise_login_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method text NOT NULL CHECK (method IN ('password','oidc','break_glass')),
    connection_id uuid REFERENCES workspace_sso_connections(id) ON DELETE CASCADE,
    workspace_id uuid REFERENCES workspaces(id) ON DELETE CASCADE,
    authenticated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT auth_session_scope_check CHECK (
        (method='oidc' AND connection_id IS NOT NULL AND workspace_id IS NOT NULL) OR
        (method='break_glass' AND connection_id IS NULL AND workspace_id IS NOT NULL) OR
        (method='password' AND connection_id IS NULL AND workspace_id IS NULL)
    )
);

CREATE INDEX enterprise_login_sessions_active_user_idx ON enterprise_login_sessions(user_id,expires_at) WHERE revoked_at IS NULL;

ALTER TABLE sso_flow_transactions ADD CONSTRAINT sso_flow_source_session_fk
    FOREIGN KEY (source_session_id) REFERENCES enterprise_login_sessions(id) ON DELETE CASCADE;

CREATE TABLE sso_exchange_codes (
    code_hash bytea PRIMARY KEY CHECK (octet_length(code_hash)=32),
    session_id uuid NOT NULL REFERENCES enterprise_login_sessions(id) ON DELETE CASCADE,
    browser_binding_hash bytea NOT NULL CHECK (octet_length(browser_binding_hash)=32),
    return_path text NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz
);

CREATE INDEX sso_exchange_codes_expiry_idx ON sso_exchange_codes(expires_at);

CREATE TABLE sso_mutation_receipts (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    request_id uuid NOT NULL,
    principal_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action text NOT NULL,
    payload_hash char(64) NOT NULL CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    response_body bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (workspace_id,request_id)
);

CREATE TABLE workspace_break_glass (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_hash bytea NOT NULL CHECK (octet_length(credential_hash)=32),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    tested_at timestamptz,
    revoked_at timestamptz,
    UNIQUE (workspace_id,user_id)
);

CREATE TABLE workspace_break_glass_audit (
    id bigserial PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('created','tested','revoked','used')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

