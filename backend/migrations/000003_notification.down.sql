----# user subscribe dao rollback
-- Add back the dropped columns
ALTER TABLE dgv_user_subscribed_dao
ADD COLUMN enable_new_proposal int not null default 1,
ADD COLUMN enable_voting_end_reminder int not null default 0;

----# dgv_notification_record rollback
-- Remove the newly added columns
ALTER TABLE dgv_notification_record
DROP COLUMN proposal_id,
DROP COLUMN vote_id;

-- Add back the dropped columns
ALTER TABLE dgv_notification_record
ADD COLUMN chain_name varchar(255) not null,
ADD COLUMN dao_name varchar(255) not null,
ADD COLUMN target_id varchar(255);

----# subscribe feature rollback
-- Drop the table created in up.sql
DROP TABLE IF EXISTS dgv_subscribed_feature;
