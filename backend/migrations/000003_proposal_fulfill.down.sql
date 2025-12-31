-- Remove fulfill fields from proposal tracking table
DROP INDEX IF EXISTS idx_dgv_proposal_tracking_fulfill_state;

ALTER TABLE dgv_proposal_tracking
DROP COLUMN IF EXISTS fulfilled,
DROP COLUMN IF EXISTS fulfilled_explain,
DROP COLUMN IF EXISTS fulfilled_at,
DROP COLUMN IF EXISTS times_fulfill,
DROP COLUMN IF EXISTS fulfill_errored;
