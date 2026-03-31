-- name: CreateConversation :one
INSERT INTO conversations (type, name, description, avatar_url, creator_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, type, name, description, avatar_url, creator_id, is_archived, last_message_at, created_at, updated_at;

-- name: GetConversationByID :one
SELECT id, type, name, description, avatar_url, creator_id, is_archived, last_message_at, created_at, updated_at
FROM conversations
WHERE id = $1;

-- name: ListConversationsByUser :many
SELECT c.id, c.type, c.name, c.description, c.avatar_url, c.creator_id, c.is_archived, c.last_message_at, c.created_at, c.updated_at,
       COALESCE(
         (SELECT COUNT(*) FROM messages m
          WHERE m.conversation_id = c.id
          AND (m.is_deleted = FALSE OR m.is_deleted IS NULL)
          AND m.created_at > COALESCE(
            (SELECT rr.last_read_at FROM read_receipts rr
             WHERE rr.conversation_id = c.id AND rr.user_id = $1),
            cp.joined_at
          )
         ), 0
       )::bigint AS unread_count
FROM conversations c
INNER JOIN conversation_participants cp ON c.id = cp.conversation_id
WHERE cp.user_id = $1 AND cp.left_at IS NULL
ORDER BY c.last_message_at DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: UpdateConversation :one
UPDATE conversations
SET name = $2, description = $3, avatar_url = $4
WHERE id = $1
RETURNING id, type, name, description, avatar_url, creator_id, is_archived, last_message_at, created_at, updated_at;

-- name: DeleteConversation :exec
DELETE FROM conversations
WHERE id = $1;

-- name: ArchiveConversation :one
UPDATE conversations
SET is_archived = TRUE
WHERE id = $1
RETURNING id, type, name, description, avatar_url, creator_id, is_archived, last_message_at, created_at, updated_at;

-- name: UnarchiveConversation :one
UPDATE conversations
SET is_archived = FALSE
WHERE id = $1
RETURNING id, type, name, description, avatar_url, creator_id, is_archived, last_message_at, created_at, updated_at;

-- name: CountConversationsByUser :one
SELECT COUNT(*) FROM conversations c
INNER JOIN conversation_participants cp ON c.id = cp.conversation_id
WHERE cp.user_id = $1 AND cp.left_at IS NULL;

-- name: SearchConversationsByUser :many
SELECT c.id, c.type, c.name, c.description, c.avatar_url, c.creator_id, c.is_archived, c.last_message_at, c.created_at, c.updated_at,
       COALESCE(
         (SELECT COUNT(*) FROM messages m
          WHERE m.conversation_id = c.id
          AND (m.is_deleted = FALSE OR m.is_deleted IS NULL)
          AND m.created_at > COALESCE(
            (SELECT rr.last_read_at FROM read_receipts rr
             WHERE rr.conversation_id = c.id AND rr.user_id = $1),
            cp.joined_at
          )
         ), 0
       )::bigint AS unread_count
FROM conversations c
INNER JOIN conversation_participants cp ON c.id = cp.conversation_id
WHERE cp.user_id = $1 AND cp.left_at IS NULL
AND (
    (c.type IN ('group', 'channel') AND c.name ILIKE '%' || @search_term || '%')
    OR
    (c.type = 'direct' AND EXISTS (
        SELECT 1 FROM conversation_participants cp2
        INNER JOIN users u ON u.id = cp2.user_id
        WHERE cp2.conversation_id = c.id
        AND cp2.user_id != $1
        AND cp2.left_at IS NULL
        AND u.username ILIKE '%' || @search_term || '%'
    ))
)
ORDER BY c.last_message_at DESC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: CountSearchConversationsByUser :one
SELECT COUNT(*) FROM conversations c
INNER JOIN conversation_participants cp ON c.id = cp.conversation_id
WHERE cp.user_id = $1 AND cp.left_at IS NULL
AND (
    (c.type IN ('group', 'channel') AND c.name ILIKE '%' || @search_term || '%')
    OR
    (c.type = 'direct' AND EXISTS (
        SELECT 1 FROM conversation_participants cp2
        INNER JOIN users u ON u.id = cp2.user_id
        WHERE cp2.conversation_id = c.id
        AND cp2.user_id != $1
        AND cp2.left_at IS NULL
        AND u.username ILIKE '%' || @search_term || '%'
    ))
);
