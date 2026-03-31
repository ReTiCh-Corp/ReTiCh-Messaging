-- name: CreateAttachment :one
INSERT INTO attachments (message_id, message_created_at, file_name, file_type, file_size, file_url, thumbnail_url, width, height, duration)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, message_id, message_created_at, file_name, file_type, file_size, file_url, thumbnail_url, width, height, duration, created_at;

-- name: ListAttachmentsByMessage :many
SELECT id, message_id, message_created_at, file_name, file_type, file_size, file_url, thumbnail_url, width, height, duration, created_at
FROM attachments
WHERE message_id = $1
ORDER BY created_at ASC;

-- name: ListAttachmentsByConversation :many
SELECT a.id, a.message_id, a.message_created_at, a.file_name, a.file_type, a.file_size, a.file_url, a.thumbnail_url, a.width, a.height, a.duration, a.created_at
FROM attachments a
INNER JOIN messages m ON a.message_id = m.id AND a.message_created_at = m.created_at
WHERE m.conversation_id = $1 AND (m.is_deleted = FALSE OR m.is_deleted IS NULL)
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAttachmentsByConversation :one
SELECT COUNT(*)
FROM attachments a
INNER JOIN messages m ON a.message_id = m.id AND a.message_created_at = m.created_at
WHERE m.conversation_id = $1 AND (m.is_deleted = FALSE OR m.is_deleted IS NULL);

-- name: GetAttachment :one
SELECT id, message_id, message_created_at, file_name, file_type, file_size, file_url, thumbnail_url, width, height, duration, created_at
FROM attachments
WHERE id = $1;
