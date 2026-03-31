-- name: CreateDeviceCode :one
INSERT INTO device_codes (
  device_code_hash,
  user_code_hash,
  client_id,
  scope,
  status,
  user_id,
  expires_at,
  poll_interval_sec,
  ip,
  user_agent
) VALUES (
  $1, $2, $3, $4,
  $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetDeviceCodeByDeviceHash :one
SELECT * FROM device_codes
WHERE device_code_hash = $1
LIMIT 1;

-- name: GetDeviceCodeByUserHash :one
SELECT * FROM device_codes
WHERE user_code_hash = $1
LIMIT 1;

-- name: UpdateDeviceCodeStatus :one
UPDATE device_codes
SET status = $2,
    user_id = COALESCE($3, user_id),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ApproveDeviceCode :one
UPDATE device_codes
SET status = 'approved',
    user_id = $2,
    updated_at = now()
WHERE id = $1
  AND status = 'pending'
RETURNING *;

-- name: TouchDeviceCodePoll :one
UPDATE device_codes
SET last_poll_at = now(),
    poll_count = poll_count + 1,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: MarkDeviceCodeConsumed :one
UPDATE device_codes
SET status = 'consumed',
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ConsumeDeviceCodeIfApproved :one
UPDATE device_codes
SET status = 'consumed',
    updated_at = now()
WHERE id = $1
  AND status = 'approved'
RETURNING *;

-- name: ExpireDeviceCodes :exec
UPDATE device_codes
SET status = 'expired',
    updated_at = now()
WHERE status IN ('pending', 'approved')
  AND expires_at <= now();
