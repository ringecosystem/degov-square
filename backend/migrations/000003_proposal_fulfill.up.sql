-- Add fulfill fields to proposal tracking table
ALTER TABLE dgv_proposal_tracking
ADD COLUMN IF NOT EXISTS fulfilled int DEFAULT 0,
ADD COLUMN IF NOT EXISTS fulfilled_explain text,
ADD COLUMN IF NOT EXISTS fulfilled_at timestamp,
ADD COLUMN IF NOT EXISTS times_fulfill int DEFAULT 0,
ADD COLUMN IF NOT EXISTS fulfill_errored int DEFAULT 0;

-- Add comments
COMMENT ON COLUMN dgv_proposal_tracking.fulfilled IS '0: not fulfilled, 1: fulfilled';
COMMENT ON COLUMN dgv_proposal_tracking.fulfilled_explain IS 'AI decision explanation JSON';
COMMENT ON COLUMN dgv_proposal_tracking.fulfilled_at IS 'Time when fulfilled';
COMMENT ON COLUMN dgv_proposal_tracking.times_fulfill IS 'Number of fulfill attempts';
COMMENT ON COLUMN dgv_proposal_tracking.fulfill_errored IS '0: no error, 1: errored after max retries';

-- Create index for efficient querying of unfulfilled proposals
CREATE INDEX IF NOT EXISTS idx_dgv_proposal_tracking_fulfill_state ON dgv_proposal_tracking (state, fulfilled, fulfill_errored) WHERE fulfilled = 0 AND fulfill_errored = 0;
