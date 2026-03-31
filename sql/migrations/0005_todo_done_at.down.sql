BEGIN;

ALTER TABLE todo_items
    DROP COLUMN done_at;

COMMIT;
