-- Add features column to dgv_dao table
-- Features is a JSON array of feature names, e.g., ["fulfill", "notify"]
ALTER TABLE dgv_dao ADD COLUMN IF NOT EXISTS features TEXT;

-- Add comment for documentation
COMMENT ON COLUMN dgv_dao.features IS 'JSON array of enabled features for this DAO, e.g., ["fulfill"]';
