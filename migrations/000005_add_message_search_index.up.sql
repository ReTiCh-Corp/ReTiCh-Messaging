-- Full-text search index on message content for fast search queries.
-- Uses 'french' config as default; queries use plainto_tsquery for simplicity.
CREATE INDEX IF NOT EXISTS idx_messages_content_search
ON messages USING gin(to_tsvector('french', COALESCE(content, '')));
