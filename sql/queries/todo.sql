-- name: CreateTodo :one
INSERT INTO todo_items (id, user_id, title, bucket)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, title, done, created_at, bucket, planned_for_day, planned_for_week;

-- name: ListTodosByUser :many
SELECT id, user_id, title, done, created_at, bucket, planned_for_day, planned_for_week
FROM todo_items
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateTodo :one
UPDATE todo_items
SET title = $3,
    done = $4
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, title, done, created_at, bucket, planned_for_day, planned_for_week;

-- name: DeleteTodo :one
DELETE FROM todo_items
WHERE id = $1 AND user_id = $2
RETURNING id;

-- name: NormalizeTodayToWeek :exec
UPDATE todo_items
SET bucket = 'week',
    planned_for_day = NULL
WHERE user_id = $1
  AND bucket = 'today'
  AND done = FALSE
  AND planned_for_day IS NOT NULL
  AND planned_for_day < CURRENT_DATE;

-- name: NormalizeWeekToLater :exec
UPDATE todo_items
SET bucket = 'later',
    planned_for_week = NULL
WHERE user_id = $1
  AND bucket = 'week'
  AND done = FALSE
  AND planned_for_week IS NOT NULL
  AND planned_for_week < date_trunc('week', CURRENT_DATE)::DATE;

-- name: UpdateTodoBucket :exec
UPDATE todo_items
SET bucket = $3,
    planned_for_day = $4,
    planned_for_week = $5
WHERE id = $1 AND user_id = $2;
