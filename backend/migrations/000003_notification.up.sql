----# user subscribe dao

-- Drop the unique indexes
DROP INDEX IF EXISTS uq_dgv_user_subscribe_uid_code ON dgv_user_subscribed_dao;

DROP INDEX IF EXISTS uq_dgv_user_subscribe_address_code ON dgv_user_subscribed_dao;

-- Drop the existing columns
ALTER TABLE dgv_user_subscribed_dao
DROP COLUMN enable_new_proposal,
DROP COLUMN enable_voting_end_reminder;

-- Add the new columns
ALTER TABLE dgv_user_subscribed_dao
ADD COLUMN feature VARCHAR(255) NOT NULL,
ADD COLUMN strategy VARCHAR(255) NOT NULL;

-- Add comments for the new columns
COMMENT ON COLUMN dgv_user_subscribed_dao.feature IS 'subscribe feature';

COMMENT ON COLUMN dgv_user_subscribed_dao.strategy IS 'subscribe strategy';


----# dgv_notification_record

alter table dgv_notification_record
drop column chain_name,
drop column dao_name,
drop column target_id;

alter table dgv_notification_record
add column proposal_id varchar(255) not null,
add column vote_id varchar(255);

COMMENT ON COLUMN dgv_notification_record.proposal_id IS 'proposal id';
COMMENT ON COLUMN dgv_notification_record.vote_id IS 'vote id';

