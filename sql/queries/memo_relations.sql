-- name: CreateMemoRelation :exec
INSERT INTO memo_relations (source_memo_id, target_memo_id, user_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: DeleteMemoRelationsBySource :exec
DELETE FROM memo_relations
WHERE source_memo_id = $1 AND user_id = $2;

-- name: ListMemoReferenceIDsBySource :many
SELECT target_memo_id
FROM memo_relations
WHERE source_memo_id = $1 AND user_id = $2
ORDER BY target_memo_id ASC;
