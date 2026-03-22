BEGIN;

ALTER TABLE todo_items
    ADD COLUMN bucket TEXT NOT NULL DEFAULT 'later',
    ADD COLUMN planned_for_day DATE,
    ADD COLUMN planned_for_week DATE;

COMMIT;
