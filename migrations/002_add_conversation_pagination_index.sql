-- Add composite index for cursor pagination on conversation list:
-- ORDER BY updated_at DESC, conversation_id DESC with owner_id filter.
-- Keep this migration idempotent for fresh databases where 001 already includes the index.
SET @idx_exists := (
    SELECT COUNT(1)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'conversations'
      AND index_name = 'idx_owner_updated_conv'
);

SET @ddl := IF(
    @idx_exists = 0,
    'ALTER TABLE conversations ADD INDEX idx_owner_updated_conv (owner_id, updated_at, conversation_id)',
    'SELECT 1'
);

PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
