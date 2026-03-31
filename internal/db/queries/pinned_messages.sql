-- name: PinMessage :one
INSERT INTO pinned_messages (conversation_id, message_id, message_created_at, pinned_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (conversation_id, message_id) DO NOTHING
RETURNING id, conversation_id, message_id, message_created_at, pinned_by, pinned_at;

-- name: UnpinMessage :exec
DELETE FROM pinned_messages
WHERE conversation_id = $1 AND message_id = $2;

-- name: ListPinnedMessages :many
SELECT id, conversation_id, message_id, message_created_at, pinned_by, pinned_at
FROM pinned_messages
WHERE conversation_id = $1
ORDER BY pinned_at DESC;

-- name: GetPinnedMessage :one
SELECT id, conversation_id, message_id, message_created_at, pinned_by, pinned_at
FROM pinned_messages
WHERE conversation_id = $1 AND message_id = $2;
