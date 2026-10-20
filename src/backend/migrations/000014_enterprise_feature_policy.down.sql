DROP INDEX IF EXISTS feature_mutation_receipts_scene;
DROP TABLE IF EXISTS feature_mutation_receipts;
DROP TRIGGER IF EXISTS enterprise_workspace_policy_create ON workspaces;
DROP FUNCTION IF EXISTS enterprise_workspace_policy_create();
DROP TRIGGER IF EXISTS enterprise_feature_policy_version ON workspace_feature_policies;
DROP FUNCTION IF EXISTS enterprise_feature_policy_version();
DROP TABLE IF EXISTS workspace_feature_policies;
ALTER TABLE share_links DROP COLUMN IF EXISTS allow_comments;
