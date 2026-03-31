BEGIN;

ALTER TABLE memos
    ALTER COLUMN content TYPE TEXT USING content::text,
    DROP COLUMN plain_text;

COMMIT;
