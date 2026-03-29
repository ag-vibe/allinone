-- name: CreateMemo :one
INSERT INTO memos (id, user_id, content, excerpt, state, archived_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, content, excerpt, state, archived_at, created_at, updated_at;

-- name: GetMemoByID :one
SELECT id, user_id, content, excerpt, state, archived_at, created_at, updated_at
FROM memos
WHERE id = $1 AND user_id = $2;

-- name: ListMemos :many
SELECT id, user_id, content, excerpt, state, archived_at, created_at, updated_at
FROM memos m
WHERE m.user_id = $1
  AND ($2::text = '' OR m.state = $2)
  AND ($3::text = '' OR m.content ILIKE '%' || $3 || '%' OR m.excerpt ILIKE '%' || $3 || '%')
  AND (
    $4::text = ''
    OR EXISTS (
      SELECT 1
      FROM memo_tags mt
      WHERE mt.memo_id = m.id
        AND mt.user_id = m.user_id
        AND mt.tag = $4
    )
  )
ORDER BY m.updated_at DESC, m.id DESC
LIMIT $5 OFFSET $6;

-- name: UpdateMemo :one
UPDATE memos
SET content = $3,
    excerpt = $4,
    state = $5,
    archived_at = $6,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, content, excerpt, state, archived_at, created_at, updated_at;

-- name: DeleteMemo :one
DELETE FROM memos
WHERE id = $1 AND user_id = $2
RETURNING id;

-- name: ListMemoBacklinks :many
SELECT m.id, m.user_id, m.content, m.excerpt, m.state, m.archived_at, m.created_at, m.updated_at
FROM memo_relations r
JOIN memos m ON m.id = r.source_memo_id AND m.user_id = r.user_id
WHERE r.user_id = $1 AND r.target_memo_id = $2
ORDER BY m.updated_at DESC, m.id DESC;
