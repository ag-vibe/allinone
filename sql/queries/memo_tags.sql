-- name: CreateMemoTag :exec
INSERT INTO memo_tags (memo_id, user_id, tag)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: DeleteMemoTagsByMemo :exec
DELETE FROM memo_tags
WHERE memo_id = $1 AND user_id = $2;

-- name: ListMemoTagsByMemo :many
SELECT tag
FROM memo_tags
WHERE memo_id = $1 AND user_id = $2
ORDER BY tag ASC;

-- name: ListTags :many
SELECT tag AS name, COUNT(*)::bigint AS count
FROM memo_tags
WHERE user_id = $1
  AND ($2::text = '' OR tag ILIKE '%' || $2 || '%')
GROUP BY tag
ORDER BY count DESC, name ASC
LIMIT $3 OFFSET $4;
