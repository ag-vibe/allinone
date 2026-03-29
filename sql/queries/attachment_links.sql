-- name: CreateAttachmentLink :exec
INSERT INTO attachment_links (attachment_id, user_id, resource_type, resource_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;

-- name: DeleteAttachmentLink :exec
DELETE FROM attachment_links
WHERE attachment_id = $1
  AND user_id = $2
  AND resource_type = $3
  AND resource_id = $4;

-- name: ListAttachmentsByResource :many
SELECT a.id, a.user_id, a.filename, a.content_type, a.size_bytes, a.storage_path, a.created_at
FROM attachment_links l
JOIN attachments a ON a.id = l.attachment_id
WHERE l.user_id = $1
  AND l.resource_type = $2
  AND l.resource_id = $3
ORDER BY l.created_at DESC;
