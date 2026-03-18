-- name: AddParticipant :one
INSERT INTO conversation_participants (conversation_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveParticipant :exec
UPDATE conversation_participants SET left_at = NOW()
WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: GetParticipant :one
SELECT * FROM conversation_participants
WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL;

-- name: ListParticipants :many
SELECT * FROM conversation_participants
WHERE conversation_id = $1 AND left_at IS NULL
ORDER BY joined_at ASC;

-- name: CountActiveParticipants :one
SELECT COUNT(*) FROM conversation_participants
WHERE conversation_id = $1 AND left_at IS NULL;

-- name: GetDirectConversationBetweenUsers :one
SELECT c.id, c.type, c.name, c.description, c.avatar_url, c.creator_id, c.is_archived, c.last_message_at, c.created_at, c.updated_at
FROM conversations c
WHERE c.type = 'direct'
AND EXISTS (
    SELECT 1 FROM conversation_participants cp1
    WHERE cp1.conversation_id = c.id AND cp1.user_id = $1 AND cp1.left_at IS NULL
)
AND EXISTS (
    SELECT 1 FROM conversation_participants cp2
    WHERE cp2.conversation_id = c.id AND cp2.user_id = $2 AND cp2.left_at IS NULL
);
