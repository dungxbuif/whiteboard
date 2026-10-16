-- Never discard acknowledged CRDT state as an ordinary migration rollback.
DO $$ BEGIN IF EXISTS(SELECT 1 FROM collab_documents) THEN RAISE EXCEPTION 'Collaboration checkpoint/drain and explicit rollback required'; END IF; END $$;
DROP TABLE collab_mutation_audit;
DROP TABLE collab_receipts;
DROP TABLE collab_updates;
DROP TABLE collab_documents;
ALTER TABLE scenes DROP COLUMN writer_mode;
