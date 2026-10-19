ALTER TABLE scenes ADD COLUMN writer_mode TEXT NOT NULL DEFAULT 'v1' CHECK(writer_mode IN ('v1','v2'));
CREATE TABLE collab_documents (
 scene_id UUID PRIMARY KEY,
 workspace_id UUID NOT NULL,
 document_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
 sequence BIGINT NOT NULL DEFAULT 0 CHECK(sequence>=0),
 snapshot_sequence BIGINT NOT NULL DEFAULT 0 CHECK(snapshot_sequence>=0 AND snapshot_sequence<=sequence),
 snapshot BYTEA NOT NULL,
 legacy_content JSONB NOT NULL,
 legacy_content_version BIGINT NOT NULL,
 snapshot_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(workspace_id,scene_id) REFERENCES scenes(workspace_id,id) ON DELETE CASCADE
);
CREATE TABLE collab_updates (
 scene_id UUID NOT NULL REFERENCES collab_documents(scene_id) ON DELETE CASCADE,
 sequence BIGINT NOT NULL CHECK(sequence>0),
 update_bytes BYTEA NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(scene_id,sequence)
);
CREATE TABLE collab_receipts (
 scene_id UUID NOT NULL REFERENCES collab_documents(scene_id) ON DELETE CASCADE,
 request_id TEXT NOT NULL CHECK(length(request_id) BETWEEN 1 AND 128),
 principal_type TEXT NOT NULL CHECK(principal_type IN ('user','public','agent')),
 principal_id TEXT NOT NULL,
 payload_hash TEXT NOT NULL CHECK(payload_hash ~ '^[0-9a-f]{64}$'),
 action TEXT NOT NULL CHECK(action IN ('scene:update','scene:command')),
 sequence BIGINT NOT NULL,
 state_vector TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(scene_id,request_id),
 UNIQUE(scene_id,sequence)
);
CREATE TABLE collab_mutation_audit (
 scene_id UUID NOT NULL,
 request_id TEXT NOT NULL,
 workspace_id UUID NOT NULL,
 principal_type TEXT NOT NULL,
 principal_id TEXT NOT NULL,
 sequence BIGINT NOT NULL,
 payload_hash TEXT NOT NULL,
 action TEXT NOT NULL,
 acl_version BIGINT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(scene_id,request_id),
 FOREIGN KEY(scene_id,request_id) REFERENCES collab_receipts(scene_id,request_id) ON DELETE CASCADE
);
