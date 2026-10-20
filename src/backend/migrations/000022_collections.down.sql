ALTER TABLE feature_mutation_receipts
  DROP CONSTRAINT IF EXISTS feature_mutation_receipts_collection_operation_check;
ALTER TABLE feature_mutation_receipts
  DROP COLUMN IF EXISTS collection_operation;
DROP TABLE IF EXISTS collection_scenes;
DROP TABLE IF EXISTS collections;
