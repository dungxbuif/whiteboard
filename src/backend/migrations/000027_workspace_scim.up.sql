CREATE TABLE workspace_scim_credentials (
 workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 token_hash BYTEA NOT NULL UNIQUE CHECK(octet_length(token_hash)=32),
 token_prefix TEXT NOT NULL CHECK(length(token_prefix) BETWEEN 8 AND 32),
 created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 revoked_at TIMESTAMPTZ
);

CREATE TABLE workspace_scim_users (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 external_id TEXT,
 user_name TEXT NOT NULL,
 display_name TEXT NOT NULL DEFAULT '',
 active BOOLEAN NOT NULL DEFAULT true,
 direct_role TEXT NOT NULL DEFAULT 'viewer' CHECK(direct_role IN ('editor','viewer')),
 retained_manual_access BOOLEAN NOT NULL DEFAULT false,
 version BIGINT NOT NULL DEFAULT 1 CHECK(version > 0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(workspace_id,id),
 UNIQUE(workspace_id,user_id),
 UNIQUE(workspace_id,external_id),
 UNIQUE(workspace_id,user_name)
);
CREATE UNIQUE INDEX workspace_scim_users_name_lower_key ON workspace_scim_users(workspace_id,lower(user_name));

CREATE TABLE workspace_scim_groups (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 external_id TEXT,
 display_name TEXT NOT NULL,
 role TEXT NOT NULL CHECK(role IN ('editor','viewer')),
 version BIGINT NOT NULL DEFAULT 1 CHECK(version > 0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(workspace_id,id),
 UNIQUE(workspace_id,external_id)
);
CREATE UNIQUE INDEX workspace_scim_groups_name_lower_key ON workspace_scim_groups(workspace_id,lower(display_name));

CREATE TABLE workspace_scim_group_members (
 workspace_id UUID NOT NULL,
 group_id UUID NOT NULL,
 scim_user_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(group_id,scim_user_id),
 FOREIGN KEY(workspace_id,group_id) REFERENCES workspace_scim_groups(workspace_id,id) ON DELETE CASCADE,
 FOREIGN KEY(workspace_id,scim_user_id) REFERENCES workspace_scim_users(workspace_id,id) ON DELETE CASCADE
);

CREATE TABLE workspace_membership_sources (
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 source_kind TEXT NOT NULL CHECK(source_kind IN ('direct','group','adopted')),
 source_id UUID NOT NULL,
 role TEXT NOT NULL CHECK(role IN ('editor','viewer')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(workspace_id,user_id,source_kind,source_id)
);
CREATE INDEX workspace_membership_sources_user_idx ON workspace_membership_sources(workspace_id,user_id);
