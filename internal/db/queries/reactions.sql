-- name: AddReaction :one
INSERT INTO message_reactions (message_id, message_created_at, user_id, emoji)
VALUES ($1, $2, $3, $4)
ON CONFLICT (message_id, user_id, emoji) DO NOTHING
RETURNING id, message_id, message_created_at, user_id, emoji, created_at;

-- name: RemoveReaction :exec
DELETE FROM message_reactions
WHERE message_id = $1 AND user_id = $2 AND emoji = $3;

-- name: ListReactionsByMessage :many
SELECT id, message_id, message_created_at, user_id, emoji, created_at
FROM message_reactions
WHERE message_id = $1
ORDER BY created_at ASC;
