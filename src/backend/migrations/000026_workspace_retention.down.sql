DELETE FROM feature_jobs WHERE kind='retention.purge';
DROP TABLE IF EXISTS retention_purge_audit;
DROP TABLE IF EXISTS retention_mutation_receipts;
DROP TABLE IF EXISTS retention_holds;
DROP TABLE IF EXISTS retention_previews;
DROP TABLE IF EXISTS workspace_retention_policies;
