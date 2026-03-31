-- name: UpsertReadReceipt :one
INSERT INTO read_receipts (conversation_id, user_id, last_read_message_id, last_read_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (conversation_id, user_id)
DO UPDATE SET last_read_message_id = EXCLUDED.last_read_message_id, last_read_at = NOW()
RETURNING id, conversation_id, user_id, last_read_message_id, last_read_at;

-- name: GetReadReceipt :one
SELECT id, conversation_id, user_id, last_read_message_id, last_read_at
FROM read_receipts
WHERE conversation_id = $1 AND user_id = $2;

-- name: ListReadReceipts :many
SELECT id, conversation_id, user_id, last_read_message_id, last_read_at
FROM read_receipts
WHERE conversation_id = $1
ORDER BY last_read_at DESC;
