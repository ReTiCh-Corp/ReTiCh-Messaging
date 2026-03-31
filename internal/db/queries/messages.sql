-- name: CreateMessage :one
INSERT INTO messages (conversation_id, sender_id, type, content, metadata, reply_to_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, conversation_id, sender_id, type, content, metadata, reply_to_id,
          is_edited, edited_at, is_deleted, deleted_at, created_at;

-- name: GetMessageByID :one
SELECT id, conversation_id, sender_id, type, content, metadata, reply_to_id,
       is_edited, edited_at, is_deleted, deleted_at, created_at
FROM messages
WHERE id = $1;

-- name: ListMessagesByConversation :many
SELECT id, conversation_id, sender_id, type, content, metadata, reply_to_id,
       is_edited, edited_at, is_deleted, deleted_at, created_at
FROM messages
WHERE conversation_id = $1 AND (is_deleted = FALSE OR is_deleted IS NULL)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountMessagesByConversation :one
SELECT COUNT(*) FROM messages
WHERE conversation_id = $1 AND (is_deleted = FALSE OR is_deleted IS NULL);

-- name: UpdateMessageContent :one
UPDATE messages SET content = $3, is_edited = TRUE, edited_at = NOW()
WHERE id = $1 AND sender_id = $2 AND (is_deleted = FALSE OR is_deleted IS NULL)
RETURNING id, conversation_id, sender_id, type, content, metadata, reply_to_id,
          is_edited, edited_at, is_deleted, deleted_at, created_at;

-- name: SearchMessages :many
SELECT id, conversation_id, sender_id, type, content, metadata, reply_to_id,
       is_edited, edited_at, is_deleted, deleted_at, created_at
FROM messages
WHERE conversation_id = $1
AND (is_deleted = FALSE OR is_deleted IS NULL)
AND to_tsvector('french', COALESCE(content, '')) @@ plainto_tsquery('french', $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountSearchMessages :one
SELECT COUNT(*) FROM messages
WHERE conversation_id = $1
AND (is_deleted = FALSE OR is_deleted IS NULL)
AND to_tsvector('french', COALESCE(content, '')) @@ plainto_tsquery('french', $2);

-- name: SoftDeleteMessage :one
UPDATE messages SET is_deleted = TRUE, deleted_at = NOW()
WHERE id = $1 AND (is_deleted = FALSE OR is_deleted IS NULL)
RETURNING id, conversation_id, sender_id, type, content, metadata, reply_to_id,
          is_edited, edited_at, is_deleted, deleted_at, created_at;
