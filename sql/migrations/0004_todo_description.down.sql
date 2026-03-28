BEGIN;

ALTER TABLE todo_items
    DROP COLUMN description;

COMMIT;
