-- Rollback Messaging Service Schema

DROP TRIGGER IF EXISTS update_last_message_trigger ON messages;
DROP TRIGGER IF EXISTS update_conversations_updated_at ON conversations;

DROP FUNCTION IF EXISTS create_messages_partition(DATE);
DROP FUNCTION IF EXISTS update_conversation_last_message();
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS pinned_messages;
DROP TABLE IF EXISTS read_receipts;
DROP TABLE IF EXISTS message_reactions;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS messages_y2026m02;
DROP TABLE IF EXISTS messages_y2026m03;
DROP TABLE IF EXISTS messages_y2026m04;
DROP TABLE IF EXISTS messages_default;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversation_participants;
DROP TABLE IF EXISTS conversations;

DROP TYPE IF EXISTS message_type;
DROP TYPE IF EXISTS conversation_type;
