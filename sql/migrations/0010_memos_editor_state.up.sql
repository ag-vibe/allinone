BEGIN;

ALTER TABLE memos
    ALTER COLUMN content TYPE JSONB USING to_jsonb(content),
    ADD COLUMN plain_text TEXT NOT NULL DEFAULT '';

UPDATE memos
SET plain_text = excerpt
WHERE plain_text = '';

COMMIT;
