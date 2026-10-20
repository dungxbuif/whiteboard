DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM scenes WHERE pending_duplicate = TRUE) THEN
        RAISE EXCEPTION 'Cannot remove pending_duplicate while unfinished scene copies exist';
    END IF;
END $$;

ALTER TABLE scenes DROP COLUMN pending_duplicate;
