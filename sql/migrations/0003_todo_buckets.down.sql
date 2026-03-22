BEGIN;

ALTER TABLE todo_items
    DROP COLUMN planned_for_week,
    DROP COLUMN planned_for_day,
    DROP COLUMN bucket;

COMMIT;
