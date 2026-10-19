CREATE TABLE workspace_invitations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 email TEXT NOT NULL CHECK (email=lower(btrim(email)) AND length(email)>0),
 role TEXT NOT NULL CHECK (role IN ('editor','viewer')),
 inviter_id UUID NOT NULL REFERENCES users(id),
 token_hash TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[a-f0-9]{64}$'),
 created_at TIMESTAMPTZ NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL CHECK (expires_at=created_at+interval '168 hours'),
 accepted_at TIMESTAMPTZ,
 revoked_at TIMESTAMPTZ,
 CHECK (accepted_at IS NULL OR revoked_at IS NULL)
);
CREATE INDEX workspace_invitations_pending ON workspace_invitations(workspace_id,expires_at) WHERE accepted_at IS NULL AND revoked_at IS NULL;
