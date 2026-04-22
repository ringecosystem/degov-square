ALTER TABLE dgv_dao ADD COLUMN IF NOT EXISTS last_tracked_block_number bigint NOT NULL DEFAULT 0;
ALTER TABLE dgv_dao ADD COLUMN IF NOT EXISTS last_tracked_proposal_id varchar(255) NOT NULL DEFAULT '';
COMMENT ON COLUMN dgv_dao.last_tracked_block_number IS 'Last tracked proposal block number (blockNumber cursor)';
COMMENT ON COLUMN dgv_dao.last_tracked_proposal_id IS 'Last tracked indexer proposal id for blockNumber tie-breaker';

-- Migrate existing data from dgv_proposal_tracking
UPDATE dgv_dao d
SET last_tracked_block_number = sub.max_block
FROM (
  SELECT dao_code, MAX(proposal_at_block) AS max_block
  FROM dgv_proposal_tracking
  GROUP BY dao_code
) sub
WHERE d.code = sub.dao_code AND sub.max_block > 0;
