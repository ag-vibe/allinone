-- name: CreateAttachment :one
INSERT INTO attachments (id, user_id, filename, content_type, size_bytes, storage_path)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, filename, content_type, size_bytes, storage_path, created_at;

-- name: GetAttachmentByID :one
SELECT id, user_id, filename, content_type, size_bytes, storage_path, created_at
FROM attachments
WHERE id = $1 AND user_id = $2;

-- name: DeleteAttachment :one
DELETE FROM attachments
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, filename, content_type, size_bytes, storage_path, created_at;
